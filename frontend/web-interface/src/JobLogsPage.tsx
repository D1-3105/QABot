import React, { useEffect, useState, useRef, useMemo } from "react";
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
    XCircle,
    Settings,
    Eye,
    EyeOff,
    Clock,
    Package,
    PlayCircle,
    StopCircle,
    Filter,
    Minimize2,
    Maximize2
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
    level?: 'info' | 'warning' | 'error' | 'success' | 'docker' | 'install' | 'test' | 'git' | 'verbose';
    operation?: string;
    isFiltered?: boolean;
    isDuplicate?: boolean;
    duplicateCount?: number;
}

interface LogFilters {
    hideVerbose: boolean;
    hideGitMessages: boolean;
    hideDuplicates: boolean;
    showOnlyErrors: boolean;
    compactMode: boolean;
    autoCollapse: boolean;
}

interface LogStats {
    total: number;
    errors: number;
    warnings: number;
    filtered: number;
    groups: number;
    startTime?: number;
    duration?: number;
}

const NOISE_PATTERNS = [
    /unable to get git ref: failed to identify reference/,
    /\[.*\]\s*unable to get git ref/,
    /warning: no-unknown-keyword/,
    /^\s*$/ // empty lines
];

const OPERATION_PATTERNS = [
    { pattern: /🐳.*docker/, type: 'docker', icon: '🐳' },
    { pattern: /📦.*download|install|npm|yarn|pip|uv/, type: 'install', icon: '📦' },
    { pattern: /🧪.*test|pytest|jest/, type: 'test', icon: '🧪' },
    { pattern: /☁.*git clone|git/, type: 'git', icon: '☁️' },
    { pattern: /⚙.*build|compile/, type: 'build', icon: '⚙️' },
    { pattern: /✅.*success|completed/, type: 'success', icon: '✅' },
    { pattern: /❌.*failed|error/, type: 'error', icon: '❌' },
    { pattern: /⚠️.*warning/, type: 'warning', icon: '⚠️' }
];

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
    const [showSettings, setShowSettings] = useState(false);
    const [filters, setFilters] = useState<LogFilters>({
        hideVerbose: true,
        hideGitMessages: true,
        hideDuplicates: true,
        showOnlyErrors: false,
        compactMode: false,
        autoCollapse: true
    });

    const containerRef = useRef<HTMLDivElement>(null);
    const abortControllerRef = useRef<AbortController | null>(null);
    const [isScrollPaused, setIsScrollPaused] = useState(false);
    const [duplicateMap, setDuplicateMap] = useState<Map<string, number>>(new Map());

    const isNoiseMessage = (content: string): boolean => {
        return NOISE_PATTERNS.some(pattern => pattern.test(content));
    };

    const detectOperation = (content: string): { type: string; icon: string } | null => {
        for (const { pattern, type, icon } of OPERATION_PATTERNS) {
            if (pattern.test(content)) {
                return { type, icon };
            }
        }
        return null;
    };

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

        const isGroupHeader = /^\[.*]\s*(⭐)/.test(content);
        const operation = detectOperation(content);

        let level: 'info' | 'warning' | 'error' | 'success' | 'docker' | 'install' | 'test' | 'git' | 'verbose' = 'info';
        const lower = content.toLowerCase();

        if (isNoiseMessage(content)) {
            level = 'verbose';
        } else if (operation) {
            level = operation.type as any;
        } else if (lower.includes('❌') || lower.includes('failure') || lower.includes('failed') || lower.includes('npm err')) {
            level = 'error';
        } else if (lower.includes('✅') || lower.includes('success') || lower.includes('✔')) {
            level = 'success';
        } else if (lower.includes('⚠️') || lower.includes('warning') || lower.includes('warn')) {
            level = 'warning';
        } else if (lower.includes('git') && !isGroupHeader) {
            level = 'git';
        }

        let groupId = '';
        if (isGroupHeader) {
            const match = content.match(/\[.*]\s*⭐\s*Run\s+(.+?)\s*$/);
            if (match) {
                groupId = match[1].trim();
            }
        }

        const isFiltered = (filters.hideVerbose && level === 'verbose') ||
                          (filters.hideGitMessages && level === 'git') ||
                          (filters.showOnlyErrors && !['error', 'warning'].includes(level));

        return {
            timestamp,
            type,
            content,
            isGroupHeader,
            level,
            groupId,
            operation: operation?.type,
            isFiltered
        };
    };

    const processLogsWithDuplicates = (rawLogs: ParsedLogLine[]): ParsedLogLine[] => {
        if (!filters.hideDuplicates) return rawLogs;

        const processed: ParsedLogLine[] = [];
        const duplicateTracker = new Map<string, { count: number; lastIndex: number }>();

        rawLogs.forEach((log, index) => {
            if (log.isGroupHeader || log.level === 'error' || log.level === 'warning') {
                processed.push(log);
                return;
            }

            const key = `${log.level}:${log.content.replace(/\d+/g, 'X')}`;
            const existing = duplicateTracker.get(key);

            if (existing && existing.count < 5) {
                existing.count++;
                existing.lastIndex = processed.length;
                processed.push({ ...log, isDuplicate: true });
            } else if (!existing) {
                duplicateTracker.set(key, { count: 1, lastIndex: processed.length });
                processed.push(log);
            } else if (existing.count >= 5) {
                if (processed[existing.lastIndex]) {
                    processed[existing.lastIndex] = {
                        ...processed[existing.lastIndex],
                        duplicateCount: existing.count
                    };
                }
            }
        });

        return processed;
    };

    const stats: LogStats = useMemo(() => {
        const firstLog = logs[0];
        const lastLog = logs[logs.length - 1];

        return {
            total: logs.length,
            errors: logs.filter(log => log.level === 'error').length,
            warnings: logs.filter(log => log.level === 'warning').length,
            filtered: logs.filter(log => log.isFiltered).length,
            groups: logs.filter(log => log.isGroupHeader).length,
            startTime: firstLog ? new Date(firstLog.timestamp).getTime() : undefined,
            duration: firstLog && lastLog ?
                new Date(lastLog.timestamp).getTime() - new Date(firstLog.timestamp).getTime() : undefined
        };
    }, [logs]);

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

    const updateFilter = (key: keyof LogFilters, value: boolean) => {
        setFilters(prev => ({ ...prev, [key]: value }));
    };

    useEffect(() => {
        if (!host || !jobId) {
            setErrorMessage('Missing host or job_id parameters');
            setConnectionStatus('error');
            return;
        }

        const url = `/api/v1/job/logs/?host=${encodeURIComponent(host)}&job_id=${encodeURIComponent(jobId)}`;
        setConnectionStatus('connecting');
        setErrorMessage('');
        setLogs([]);

        const abortController = new AbortController();
        abortControllerRef.current = abortController;

        const fetchLogs = async () => {
            try {
                const response = await fetch(url, {
                    headers: { 'Accept': 'text/event-stream' },
                    signal: abortController.signal,
                });

                if (!response.ok) {
                    setConnectionStatus('error');
                    setErrorMessage(`HTTP ${response.status}: ${response.statusText}`);
                    return;
                }

                setConnectionStatus('connected');
                const reader = response.body?.getReader();
                const decoder = new TextDecoder();

                if (!reader) {
                    setConnectionStatus('error');
                    setErrorMessage('Failed to read body!');
                    return;
                }

                let buffer = '';
                let currentGroupId = '';

                while (true) {
                    const { done, value } = await reader.read();

                    if (done) {
                        setConnectionStatus('completed');
                        break;
                    }

                    buffer += decoder.decode(value, { stream: true });
                    const lines = buffer.split('\n');
                    buffer = lines.pop() || '';

                    for (const line of lines) {
                        if (line.trim() === '' || !line.startsWith('data: ')) continue;

                        const dataStr = line.substring(6);
                        try {
                            const data: LogEntry = JSON.parse(dataStr);
                            if (data && typeof data === 'object' && 'line' in data) {
                                let parsed = parseLogLine(data);

                                if (parsed.isGroupHeader) {
                                    currentGroupId = parsed.groupId;
                                    if (filters.autoCollapse && collapsedGroups.has(parsed.groupId)) {
                                        // Auto-expand new groups
                                        setCollapsedGroups(prev => {
                                            const newSet = new Set(prev);
                                            newSet.delete(parsed.groupId);
                                            return newSet;
                                        });
                                    }
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
            } catch (error: unknown) {
                if (error instanceof DOMException && error.name === 'AbortError') return;

                const message = error instanceof Error ? error.message : String(error);
                setConnectionStatus('error');
                setErrorMessage(message || 'Failed to fetch logs');
            }
        };

        fetchLogs();

        return () => {
            if (abortControllerRef.current) {
                abortControllerRef.current.abort();
            }
            abortControllerRef.current = null;
        };
    }, [host, jobId]);

    useEffect(() => {
        if (containerRef.current && !isScrollPaused) {
            containerRef.current.scrollTop = containerRef.current.scrollHeight;
        }
    }, [logs, isScrollPaused]);

    const processedLogs = useMemo(() => {
        let filtered = logs;

        if (searchTerm) {
            filtered = filtered.filter(log =>
                log.content.toLowerCase().includes(searchTerm.toLowerCase())
            );
        }

        return processLogsWithDuplicates(filtered).filter(log => !log.isFiltered);
    }, [logs, searchTerm, filters]);

    const handleRefresh = () => window.location.reload();

    const handleDownload = () => {
        const logText = processedLogs
            .map(log => `${log.timestamp} [${log.type.toUpperCase()}] ${log.content}`)
            .join('\n');
        const blob = new Blob([logText], { type: 'text/plain' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `job-logs-${jobId?.slice(0, 8)}-filtered.txt`;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
    };

    const getStatusIcon = () => {
        switch (connectionStatus) {
            case 'connected':
                return <Activity className="animate-pulse" style={{ width: 16, height: 16, color: '#f59e0b' }} />;
            case 'completed':
                return <CheckCircle style={{ width: 16, height: 16, color: '#10b981' }} />;
            case 'error':
                return <AlertCircle style={{ width: 16, height: 16, color: '#ef4444' }} />;
            default:
                return <div className="animate-spin w-4 h-4 border-2 border-gray-300 border-t-blue-600 rounded-full" />;
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
            }
        } catch (err) {
            console.error('Cancel error:', err);
        }
    };

    const getLevelIcon = (level: string) => {
        switch (level) {
            case 'docker': return '🐳';
            case 'install': return '📦';
            case 'test': return '🧪';
            case 'git': return '☁️';
            case 'build': return '⚙️';
            case 'error': return '❌';
            case 'warning': return '⚠️';
            case 'success': return '✅';
            case 'verbose': return '🔍';
            default: return '📝';
        }
    };

    const getLevelColor = (level: string) => {
        switch (level) {
            case 'error': return 'text-red-400';
            case 'warning': return 'text-yellow-400';
            case 'success': return 'text-green-400';
            case 'docker': return 'text-blue-400';
            case 'install': return 'text-purple-400';
            case 'test': return 'text-cyan-400';
            case 'git': return 'text-orange-400';
            case 'verbose': return 'text-gray-500';
            default: return 'text-gray-300';
        }
    };

    return (
        <div className="container">
            {/* Enhanced Header */}
            <div className="header">
                <div className="header-overlay" />
                <div className="header-content">
                    <div className="header-left">
                        <div className="header-info">
                            <div className="icon-container">
                                <Terminal style={{ width: 24, height: 24, color: 'white' }} />
                            </div>
                            <div>
                                <h1 className="title">Job Logs</h1>
                                <div className="status-container">
                                    <div className="status-badge" style={getStatusStyle()}>
                                        {getStatusIcon()}
                                        <span className="status-text">{connectionStatus}</span>
                                    </div>
                                    <div className="stats-badges">
                                        <span className="stat-badge">
                                            📊 {stats.total} lines
                                        </span>
                                        {stats.errors > 0 && (
                                            <span className="stat-badge error">
                                                ❌ {stats.errors} errors
                                            </span>
                                        )}
                                        {stats.warnings > 0 && (
                                            <span className="stat-badge warning">
                                                ⚠️ {stats.warnings} warnings
                                            </span>
                                        )}
                                        {stats.filtered > 0 && (
                                            <span className="stat-badge filtered">
                                                🔍 {stats.filtered} filtered
                                            </span>
                                        )}
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>

                    <div className="header-right">
                        <div className="search-container">
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
                            <button
                                onClick={() => setShowSettings(!showSettings)}
                                className={`settings-button ${showSettings ? 'active' : ''}`}
                            >
                                <Settings className="icon" />
                                Filters
                            </button>

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
                                Cancel
                            </button>
                        </div>
                    </div>
                </div>

                {/* Filter Panel */}
                {showSettings && (
                    <div className="filter-panel">
                        <div className="filter-section">
                            <h3>Display Options</h3>
                            <div className="filter-options">
                                <label className="filter-option">
                                    <input
                                        type="checkbox"
                                        checked={filters.hideVerbose}
                                        onChange={(e) => updateFilter('hideVerbose', e.target.checked)}
                                    />
                                    <span>Hide verbose messages</span>
                                </label>
                                <label className="filter-option">
                                    <input
                                        type="checkbox"
                                        checked={filters.hideGitMessages}
                                        onChange={(e) => updateFilter('hideGitMessages', e.target.checked)}
                                    />
                                    <span>Hide git reference warnings</span>
                                </label>
                                <label className="filter-option">
                                    <input
                                        type="checkbox"
                                        checked={filters.hideDuplicates}
                                        onChange={(e) => updateFilter('hideDuplicates', e.target.checked)}
                                    />
                                    <span>Group duplicate messages</span>
                                </label>
                                <label className="filter-option">
                                    <input
                                        type="checkbox"
                                        checked={filters.showOnlyErrors}
                                        onChange={(e) => updateFilter('showOnlyErrors', e.target.checked)}
                                    />
                                    <span>Show only errors & warnings</span>
                                </label>
                                <label className="filter-option">
                                    <input
                                        type="checkbox"
                                        checked={filters.compactMode}
                                        onChange={(e) => updateFilter('compactMode', e.target.checked)}
                                    />
                                    <span>Compact mode</span>
                                </label>
                                <label className="filter-option">
                                    <input
                                        type="checkbox"
                                        checked={filters.autoCollapse}
                                        onChange={(e) => updateFilter('autoCollapse', e.target.checked)}
                                    />
                                    <span>Auto-collapse completed groups</span>
                                </label>
                            </div>
                        </div>

                        <div className="filter-section">
                            <h3>Scroll Control</h3>
                            <button
                                onClick={() => setIsScrollPaused(!isScrollPaused)}
                                className={`scroll-control ${isScrollPaused ? 'paused' : 'active'}`}
                            >
                                {isScrollPaused ? <PlayCircle className="icon" /> : <StopCircle className="icon" />}
                                {isScrollPaused ? 'Resume Auto-scroll' : 'Pause Auto-scroll'}
                            </button>
                        </div>
                    </div>
                )}
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
                    <span className="toast-icon">✅</span>
                    <span className="toast-message">Job canceled successfully</span>
                </div>
            )}

            {/* Enhanced Logs Display */}
            <div className="logs-wrapper">
                <div className="logs-container">
                    <div className="terminal-header">
                        <div className="terminal-buttons">
                            <div className="terminal-button" style={{ backgroundColor: '#ff5f56' }} />
                            <div className="terminal-button" style={{ backgroundColor: '#ffbd2e' }} />
                            <div className="terminal-button" style={{ backgroundColor: '#27ca3f' }} />
                        </div>
                        <div className="terminal-title">
                            {host} • {jobId} • {processedLogs.length} lines
                        </div>
                        <div className="terminal-controls">
                            {filters.compactMode ? (
                                <Maximize2
                                    className="control-icon"
                                    onClick={() => updateFilter('compactMode', false)}
                                />
                            ) : (
                                <Minimize2
                                    className="control-icon"
                                    onClick={() => updateFilter('compactMode', true)}
                                />
                            )}
                        </div>
                    </div>

                    <div
                        ref={containerRef}
                        className={`logs-content ${filters.compactMode ? 'compact' : ''}`}
                        onScroll={(e) => {
                            const { scrollTop, scrollHeight, clientHeight } = e.currentTarget;
                            const isAtBottom = scrollHeight - scrollTop <= clientHeight + 50;
                            setIsScrollPaused(!isAtBottom);
                        }}
                    >
                        {processedLogs.length === 0 ? (
                            <div className="empty-state">
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
                        ) : (
                            <div className="logs-list">
                                {processedLogs.map((log, index) => {
                                    if (!log.isGroupHeader && log.groupId && collapsedGroups.has(log.groupId)) {
                                        return null;
                                    }

                                    return (
                                        <div key={index} className={`log-entry ${log.level} ${filters.compactMode ? 'compact' : ''}`}>
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
                                                    <span className="group-icon">{getLevelIcon(log.level || 'info')}</span>
                                                    <span className="timestamp">{log.timestamp}</span>
                                                    <span className={`log-type ${log.type === 'stderr' ? 'stderr' : 'stdout'}`}>
                                                        {log.type.toUpperCase()}
                                                    </span>
                                                    <span className={`group-content ${getLevelColor(log.level || 'info')}`}>
                                                        {log.content}
                                                    </span>
                                                </div>
                                            ) : (
                                                <div className={`log-line ${getLevelColor(log.level || 'info')}`}>
                                                    {!filters.compactMode && (
                                                        <>
                                                            <span className="log-icon">{getLevelIcon(log.level || 'info')}</span>
                                                            <span className="log-timestamp">{log.timestamp}</span>
                                                        </>
                                                    )}
                                                    <span className="log-content">
                                                        {log.content}
                                                        {log.duplicateCount && (
                                                            <span className="duplicate-badge">
                                                                +{log.duplicateCount} similar
                                                            </span>
                                                        )}
                                                    </span>
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
    );
};

export default JobLogsPage;