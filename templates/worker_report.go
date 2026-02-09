package templates

type WorkerReportContext struct {
	MultilineGithubComment
	ReportText string
}

func NewWorkerReportContext(initialCommand string, botInitialReply string, textReport string) *WorkerReportContext {
	tmpInit()
	return &WorkerReportContext{
		MultilineGithubComment: NewMultilineGithubComment(
			[]string{initialCommand, botInitialReply}, templateEnv.WorkerReportTemplate,
		),
		ReportText: textReport,
	}
}

func (wr *WorkerReportContext) GenText() (string, error) {
	return GenTextFromTemplate(wr.tmplFile, wr)
}
