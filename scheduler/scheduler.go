package scheduler

type Scheduler interface {
	SelectCandidatteNodes()
	Score()
	Pick()
}
