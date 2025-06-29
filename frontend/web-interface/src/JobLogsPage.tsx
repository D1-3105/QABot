import React, { useEffect, useState, useRef } from "react";
import { useSearchParams } from "react-router-dom";
import {
    RotateCcw,
    Download,
    Search,
    ChevronRight,
    ChevronDown,
    Terminal,
    Activity,
    AlertCircle,
    CheckCircle,
    XCircle
} from "lucide-react";

interface LogEntry {
    timestamp: number;
    type: number;
    line: string;
}

interface ParsedLogLine {
    timestamp: string;
    type: 'stdout' | 'stderr';
    content: string;
    isGroupHeader: boolean;
    isCollapsed?: boolean;
    groupId: string;
    level?: 'info' | 'warning' | 'error' | 'success';
}

const JobLogsPage = () => {
    const [searchParams] = useSearchParams();
    const host = searchParams.get("host");
    const jobId = searchParams.get("job_id");

    const [logs, setLogs] = useState<ParsedLogLine[]>([]);
    const [canceled, setCanceled] = useState(false);
    const [connectionStatus, setConnectionStatus] = useState<'connecting' | 'connected' | 'error' | 'completed'>('connecting');
    const [errorMessage, setErrorMessage] = useState<string>('');
    const [searchTerm, setSearchTerm] = useState<string>('');
    const [collapsedGroups, setCollapsedGroups] = useState<Set<string>>(new Set());
    const containerRef = useRef<HTMLDivElement>(null);
    const abortControllerRef = useRef<AbortController | null>(null);

    const parseLogLine = (data: LogEntry): ParsedLogLine => {
        const timestamp = new Date(data.timestamp * 1000).toLocaleTimeString('en-US', {
            hour12: false,
            hour: '2-digit',
            minute: '2-digit',
            second: '2-digit'
        });

        const type = data.type === 1 ? 'stdout' : 'stderr';

        let content = data.line;
        try {
            const decoder = new TextDecoder('utf-8');
            const bytes = new TextEncoder().encode(content);
            content = decoder.decode(bytes);
        } catch (_) {
            // fallback to raw content
        }

        const isGroupHeader = /^\[CI\/test]\s*(⭐)/.test(content);

        let level: 'info' | 'warning' | 'error' | 'success' = 'info';
        const lower = content.toLowerCase();

        if (lower.includes('❌') || lower.includes('failure') || lower.includes('failed') || lower.includes('npm err')) {
            level = 'error';
        } else if (lower.includes('✅') || lower.includes('success') || lower.includes('✔')) {
            level = 'success';
        } else if (lower.includes('⚠️') || lower.includes('warning') || lower.includes('warn')) {
            level = 'warning';
        }

        let groupId = '';
        if (isGroupHeader) {
            const match = content.match(/\[.*]\s*⭐\s*Run\s+(.+?)\s*$/);
            if (match) {
                groupId = match[1].trim();
            }
        }

        return {
            timestamp,
            type,
            content,
            isGroupHeader,
            level,
            groupId
        };
    };

    const toggleGroup = (groupId: string) => {
        setCollapsedGroups(prev => {
            const newSet = new Set(prev);
            if (newSet.has(groupId)) {
                newSet.delete(groupId);
            } else {
                newSet.add(groupId);
            }
            return newSet;
        });
    };

    useEffect(() => {
        if (!host || !jobId) {
            setErrorMessage('Missing host or job_id parameters');
            setConnectionStatus('error');
            return;
        }

        const url = `/api/v1/job/logs/?host=${encodeURIComponent(host)}&job_id=${encodeURIComponent(jobId)}`;

        console.log('Fetching logs from:', url);
        setConnectionStatus('connecting');
        setErrorMessage('');
        setLogs([]);

        const abortController = new AbortController();
        abortControllerRef.current = abortController;

        const fetchLogs = async () => {
            try {
                const response = await fetch(url, {
                    headers: {
                        'Accept': 'text/event-stream',
                    },
                    signal: abortController.signal,
                });

                if (!response.ok) {
                    console.error(`HTTP ${response.status}: ${response.statusText}`);
                    setConnectionStatus('error');
                    setErrorMessage(`HTTP ${response.status}: ${response.statusText}`);
                    return;
                }

                setConnectionStatus('connected');

                const reader = response.body?.getReader();
                const decoder = new TextDecoder();

                if (!reader) {
                    console.error('No response body reader available');
                    setConnectionStatus('error');
                    setErrorMessage(`failed to read body!`);
                    return;
                }

                let buffer = '';

                let currentGroupId = '';

                while (true) {
                    const { done, value } = await reader.read();

                    if (done) {
                        console.log('Stream completed');
                        setConnectionStatus('completed');
                        break;
                    }

                    buffer += decoder.decode(value, { stream: true });
                    const lines = buffer.split('\n');
                    buffer = lines.pop() || '';

                    for (const line of lines) {
                        if (line.trim() === '') continue;

                        if (line.startsWith('data: ')) {
                            const dataStr = line.substring(6);
                            try {
                                const data: LogEntry = JSON.parse(dataStr);

                                if (data && typeof data === 'object' && 'line' in data) {
                                    let parsed = parseLogLine(data);
                                    console.log('parsed', parsed);
                                    if (parsed.isGroupHeader) {
                                        currentGroupId = parsed.groupId;
                                    } else if (currentGroupId) {
                                        parsed.groupId = currentGroupId;
                                    }

                                    setLogs(prevLogs => [...prevLogs, parsed]);
                                }
                            } catch (parseError) {
                                console.error('Failed to parse SSE data:', dataStr, parseError);
                            }
                        }
                    }
                }

            } catch (error: unknown) {
                if (error instanceof DOMException && error.name === 'AbortError') {
                    console.log('Fetch aborted');
                    return;
                }

                const message = error instanceof Error ? error.message : String(error);
                console.error('Fetch error:', error);
                setConnectionStatus('error');
                setErrorMessage(message || 'Failed to fetch logs');
            }
        };

        fetchLogs();

        return () => {
            console.log('Aborting fetch request');
            if (abortControllerRef.current) {
                abortControllerRef.current.abort();
            }
            abortControllerRef.current = null;
        };
    }, [host, jobId]);

    useEffect(() => {
        if (containerRef.current) {
            containerRef.current.scrollTop = containerRef.current.scrollHeight;
        }
    }, [logs]);

    const filteredLogs = logs.filter(log =>
        searchTerm === '' || log.content.toLowerCase().includes(searchTerm.toLowerCase())
    );

    const handleRefresh = () => {
        window.location.reload();
    };

    const handleDownload = () => {
        const logText = logs.map(log => `${log.timestamp} [${log.type.toUpperCase()}] ${log.content}`).join('\n');
        const blob = new Blob([logText], { type: 'text/plain' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `job-logs-${jobId?.slice(0, 8)}.txt`;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
    };

    const getStatusIcon = () => {
        switch (connectionStatus) {
            case 'connected':
                return <Activity style={{ width: 16, height: 16, color: '#f59e0b' }} className="pulse" />;
            case 'completed':
                return <CheckCircle style={{ width: 16, height: 16, color: '#10b981' }} />;
            case 'error':
                return <AlertCircle style={{ width: 16, height: 16, color: '#ef4444' }} />;
            default:
                return <div className="spinner" />;
        }
    };

    const getStatusStyle = () => {
        switch (connectionStatus) {
            case 'connected':
                return {
                    background: 'linear-gradient(135deg, rgba(251, 191, 36, 0.2), rgba(245, 158, 11, 0.2))',
                    borderColor: 'rgba(245, 158, 11, 0.3)',
                    color: '#92400e'
                };
            case 'completed':
                return {
                    background: 'linear-gradient(135deg, rgba(16, 185, 129, 0.2), rgba(5, 150, 105, 0.2))',
                    borderColor: 'rgba(16, 185, 129, 0.3)',
                    color: '#047857'
                };
            case 'error':
                return {
                    background: 'linear-gradient(135deg, rgba(239, 68, 68, 0.2), rgba(220, 38, 38, 0.2))',
                    borderColor: 'rgba(239, 68, 68, 0.3)',
                    color: '#b91c1c'
                };
            default:
                return {
                    background: 'linear-gradient(135deg, rgba(107, 114, 128, 0.2), rgba(75, 85, 99, 0.2))',
                    borderColor: 'rgba(107, 114, 128, 0.3)',
                    color: '#374151'
                };
        }
    };



    const cancelJob = async () => {
        try {
            const url = `/api/v1/job/cancel/?host=${host}&job_id=${jobId}`;
            const res = await fetch(url, { method: 'PATCH' });

            if (res.status === 204) {
                setCanceled(true);
            } else {
                console.error('Cancel failed:', res);
            }
        } catch (err) {
            console.error('Cancel error:', err);
        }
    };

    return (
        <div className="container">
            {/* Header */}
            <div className="header">
                <div className="header-overlay" />
                <div className="header-content">
                    <div className="header-left">
                        <div className="header-top">
                            <div className="status-container">
                                <div className={`status-badge ${getStatusStyle()}`}>
                                    {getStatusIcon()}
                                    <span className="status-text">{connectionStatus}</span>
                                </div>
                                <div className="line-count">
                                    {filteredLogs.length} {filteredLogs.length === 1 ? 'line' : 'lines'}
                                </div>
                            </div>
                        </div>
                    </div>

                    <div className="header-right">
                        <div className="search-container">
                            <div className="search-glow" />
                            <div className="search-input-container">
                                <Search className="search-icon" />
                                <input
                                    type="text"
                                    placeholder="Search logs..."
                                    value={searchTerm}
                                    onChange={(e) => setSearchTerm(e.target.value)}
                                    className="search-input"
                                />
                            </div>
                        </div>

                        <div className="button-group">
                            <button onClick={handleDownload} className="export-button">
                                <Download className="icon" />
                                Export
                            </button>

                            <button onClick={handleRefresh} className="refresh-button">
                                <RotateCcw className="icon" />
                                Refresh
                            </button>

                            <button onClick={cancelJob} className="cancel-button">
                                <XCircle className="icon" />
                                Cancel Job
                            </button>
                        </div>
                    </div>
                </div>
            </div>

            {errorMessage && connectionStatus === 'error' && (
                <div className="error-container">
                    <div className="error-content">
                        <div className="error-icon">
                            <AlertCircle className="error-icon-svg" />
                        </div>
                        <div>
                            <h3 className="error-title">Connection Error</h3>
                            <p className="error-message">{errorMessage}</p>
                        </div>
                    </div>
                </div>
            )}

            {canceled && (
                <div className="toast toast-success">
                    <div className="toast-content">
                        <span className="toast-icon">✅</span>
                        <span className="toast-message">Job canceled successfully.</span>
                    </div>
                </div>
            )}

            <div className="logs-wrapper">
                <div className="logs-container">
                    <div className="logs-inner">
                        <div className="terminal-header">
                            <div className="terminal-buttons">
                                <div className="terminal-button red" />
                                <div className="terminal-button yellow" />
                                <div className="terminal-button green" />
                            </div>
                            <div className="terminal-title">
                                {host} • {jobId}
                            </div>
                        </div>

                        <div ref={containerRef} className="logs-content">
                            {filteredLogs.length === 0 ? (
                                <div className="empty-state">
                                    <div className="empty-state-content">
                                        {connectionStatus === 'connecting' ? (
                                            <div className="loading-state">
                                                <div className="spinner-large" />
                                                <p className="loading-text">Loading logs...</p>
                                            </div>
                                        ) : (
                                            <div className="no-logs-state">
                                                <Terminal className="no-logs-icon" />
                                                <p className="no-logs-text">
                                                    {connectionStatus === 'error' ? 'Failed to load logs' : 'No logs to display'}
                                                </p>
                                            </div>
                                        )}
                                    </div>
                                </div>
                            ) : (
                                <div className="logs-list">
                                    {filteredLogs.map((log, index) => {
                                        if (!log.isGroupHeader && log.groupId && collapsedGroups.has(log.groupId)) {
                                            return null;
                                        }

                                        return (
                                            <div key={index} className="log-group fadeIn">
                                                {log.isGroupHeader ? (
                                                    <div
                                                        className="group-header"
                                                        onClick={() => log.groupId && toggleGroup(log.groupId)}
                                                    >
                                                        {log.groupId && (
                                                            <div className="chevron">
                                                                {collapsedGroups.has(log.groupId) ? (
                                                                    <ChevronRight className="chevron-icon" />
                                                                ) : (
                                                                    <ChevronDown className="chevron-icon" />
                                                                )}
                                                            </div>
                                                        )}
                                                        <span className="timestamp">{log.timestamp}</span>
                                                        <span className={`log-type ${log.type === 'stderr' ? 'stderr' : 'stdout'}`}>
                                                            {log.type.toUpperCase()}
                                                        </span>
                                                        <span className={`group-content level-${log.level}`}>
                                                            {log.content}
                                                        </span>
                                                    </div>
                                                ) : (
                                                    <div className={`log-line level-${log.level}`}>
                                                        {log.content}
                                                    </div>
                                                )}
                                            </div>
                                        );
                                    })}
                                </div>
                            )}
                        </div>
                    </div>
                </div>
            </div>
        </div>
    );
};

export default JobLogsPage;