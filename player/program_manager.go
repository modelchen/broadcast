package player

import (
	"github.com/robfig/cron"
)

type ProgramManager struct {
	cronJob *cron.Cron
}

func NewProgramManager() *ProgramManager {
	manager := &ProgramManager{
		cronJob: cron.New(),
	}

	return manager
}

func (m *ProgramManager) Start() {
	m.cronJob.Start()
}

func (m *ProgramManager) Stop() {
	m.cronJob.Stop()
}

func (m *ProgramManager) Clear() {
	m.cronJob.Stop()
	m.cronJob = cron.New()
}

func (m *ProgramManager) AddProgram(scheduleExpr string, pgm *Program, actFunc ProgramActiveFunc) {
	job := NewProgramJob(pgm, actFunc)
	_ = m.cronJob.AddJob(scheduleExpr, job)
}

func (m *ProgramManager) AddFunc(scheduleExpr string, cmd func()) {
	_ = m.cronJob.AddFunc(scheduleExpr, cmd)
}

type ProgramActiveFunc func(program *Program)

type ProgramJob struct {
	program    *Program
	activeFunc ProgramActiveFunc
}

func NewProgramJob(pgm *Program, actFunc ProgramActiveFunc) (job *ProgramJob) {
	job = &ProgramJob{
		program:    pgm,
		activeFunc: actFunc,
	}
	return job
}

func (pj *ProgramJob) Run() {
	if pj.activeFunc != nil {
		pj.activeFunc(pj.program)
	}
}
