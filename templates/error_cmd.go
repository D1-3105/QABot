package templates

type ErrorResultContext struct {
	MultilineGithubComment
	ErrorText string
}

func NewErrorResultContext(errorText string) *ErrorResultContext {
	tmpInit()
	return &ErrorResultContext{
		MultilineGithubComment: NewMultilineGithubComment(make([]string, 0), templateEnv.ErrorTemplate),
		ErrorText:              errorText,
	}
}

func (e *ErrorResultContext) GenText() (string, error) {
	return GenTextFromTemplate(e.tmplFile, e)
}
