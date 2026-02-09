package worker_report

import (
	"ActQABot/conf"
	"ActQABot/pkg/github/gh_api"
	"ActQABot/templates"
	"context"
	"github.com/golang/glog"
	"sync"
)

type jobReportsConsumerState struct {
	mu       sync.Locker
	perIdent map[string]chan any
}

func JobReportsConsumer(ctx context.Context, jobReportEventLoop <-chan *JobReportEvent) {
	perJobExec := jobReportsConsumerState{perIdent: make(map[string]chan any), mu: &sync.Mutex{}}
	defer glog.V(1).Infof("JobReportsConsumer end.")
	for {
		glog.V(1).Infof("JobReportsConsumer start...")
		select {
		case jobReportEvent := <-jobReportEventLoop:
			{
				report := jobReportEvent.Report
				glog.V(1).Infof("JobReportsConsumer received job report: %s", report.JobId)
				// single task per ID
				// Do not block the whole loop waiting for the semaphore inside the lock.
				getSyncChannel := func() chan any {
					glog.V(2).Infof("JobReportsConsumer is waiting for the map lock")
					perJobExec.mu.Lock()
					defer perJobExec.mu.Unlock()
					defer glog.V(2).Infof("JobReportsConsumer released the map lock")

					glog.V(1).Infof("JobReportsConsumer Received jobReportEvent jobReportEvent: %s", report.JobId)
					syncExec, found := perJobExec.perIdent[report.JobId]
					if !found {
						syncExec = make(chan any, 1)
						perJobExec.perIdent[report.JobId] = syncExec
					}
					return syncExec
				}

				syncExec := getSyncChannel()

				// async execution per channel, but sync per ID
				go func(syncExec chan any, jobId string) {
					// Acquire semaphore outside of the map mutex to avoid blocking other jobs
					glog.V(1).Infof("JobReportsConsumer is waiting for semaphore >>%s<<...", jobId)
					select {
					case syncExec <- struct{}{}:
						glog.V(1).Infof("JobReportsConsumer acquired the semaphore >>%s<<", jobId)
					case <-ctx.Done():
						return
					}

					// unlock current jobID
					defer func() {
						glog.V(2).Infof("defer JobReportsConsumer released the map lock")
						perJobExec.mu.Lock()
						<-syncExec
						// This prevents memory leaks while avoiding "stealing" channels from new arrivals
						if len(syncExec) == 0 {
							delete(perJobExec.perIdent, jobId)
						}

						glog.V(1).Infof("defer JobReportsConsumer released the semaphore >>%s<<", jobId)
						perJobExec.mu.Unlock()
					}()

					// retrieve github meta
					job, err := RetrieveGithubJobMetaFunc(ctx, report.JobId)
					if err != nil {
						glog.Errorf("JobReportsConsumer - error during retrieval: %v", err)

						// nack on error
						jobReportEvent.Finish.Do(
							func() {
								if err := jobReportEvent.Nack(ctx); err != nil {
									glog.Errorf("JobReportsConsumer - error during nack of %s: %v", report.JobId, err)
								}
							},
						)
						return
					}
					if job == nil || job.AnswerCommentBody == nil {
						glog.Errorf("JobReportsConsumer - job %v does not exist", report.JobId)
						jobReportEvent.Finish.Do(
							func() {
								if err := jobReportEvent.Nack(ctx); err != nil {
									glog.Errorf("JobReportsConsumer - error during nack of %s: %v", report.JobId, err)
								}
							},
						)
						return
					}
					// authorize github
					tok, err := gh_api.Authorize(
						conf.GithubEnvironment, job.Owner, job.Repository,
					)

					// generate response
					generated, err := templates.NewWorkerReportContext(
						job.Body, *job.AnswerCommentBody, report.JobReportText,
					).GenText()
					if err != nil {
						glog.Errorf("JobReportsConsumer - error during token generation: %v", err)
						jobReportEvent.Finish.Do(
							func() {
								if err := jobReportEvent.Nack(ctx); err != nil {
									glog.Errorf("JobReportsConsumer - error during nack of %s: %v", report.JobId, err)
								}
							},
						)
						return
					}

					// post comment
					if err = gh_api.PostIssueCommentFunc(
						&gh_api.BotResponse{
							Owner:       job.Owner,
							Repo:        job.Repository,
							IssueNumber: job.IssueId,
							Text:        generated,
						},
						*tok.Token,
					); err != nil {
						glog.Errorf("JobReportsConsumer - error during posting issue comment: %v", err)
						jobReportEvent.Finish.Do(
							func() {
								if err := jobReportEvent.Nack(ctx); err != nil {
									glog.Errorf("JobReportsConsumer - error during nack of %s: %v", report.JobId, err)
								}
							},
						)
						return
					}

					// ack
					jobReportEvent.Finish.Do(
						func() {
							if err = jobReportEvent.Ack(ctx); err != nil {
								glog.Errorf("JobReportsConsumer - error during ack of %s: %v", report.JobId, err)
								return
							}
						},
					)
					// we don't delete the job because we might have some additional worker reports
					// let's just trust the Lease.
				}(syncExec, report.JobId)
				continue
			}

		case <-ctx.Done():
			return
		}
	}
}
