//go:build windows

package main

import (
	"fmt"
	"sync"
	"unsafe"
)

var (
	createJobObject    = kernel32.NewProc("CreateJobObjectW")
	setJobInformation  = kernel32.NewProc("SetInformationJobObject")
	assignProcessToJob = kernel32.NewProc("AssignProcessToJobObject")
	terminateJobObject = kernel32.NewProc("TerminateJobObject")
	openProcess        = kernel32.NewProc("OpenProcess")
)

type jobBasicLimits struct {
	ProcessTime, JobTime           int64
	Flags                          uint32
	MinWorkingSet, MaxWorkingSet   uintptr
	ActiveProcessLimit             uint32
	Affinity                       uintptr
	PriorityClass, SchedulingClass uint32
}

type jobExtendedLimits struct {
	Basic                                                                jobBasicLimits
	IOCounters                                                           [6]uint64
	ProcessMemoryLimit, JobMemoryLimit, PeakProcessMemory, PeakJobMemory uintptr
}

type workerJob struct {
	mu     sync.Mutex
	handle uintptr
}

func newWorkerJob() (*workerJob, error) {
	handle, _, err := createJobObject.Call(0, 0)
	if handle == 0 {
		return nil, fmt.Errorf("无法创建输入服务任务：%v", err)
	}
	j := &workerJob{handle: handle}
	limits := jobExtendedLimits{Basic: jobBasicLimits{Flags: 0x2000}} // JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE.
	ok, _, err := setJobInformation.Call(handle, 9, uintptr(unsafe.Pointer(&limits)), unsafe.Sizeof(limits))
	if ok == 0 {
		j.close()
		return nil, fmt.Errorf("无法设置输入服务退出保护：%v", err)
	}
	return j, nil
}

func (j *workerJob) assign(pid int) error {
	process, _, err := openProcess.Call(0x0100|0x0001, 0, uintptr(pid)) // SET_QUOTA | TERMINATE.
	if process == 0 {
		return fmt.Errorf("无法管理输入服务：%v", err)
	}
	defer closeHandle.Call(process)
	ok, _, err := assignProcessToJob.Call(j.handle, process)
	if ok == 0 {
		return fmt.Errorf("无法绑定输入服务退出保护：%v", err)
	}
	return nil
}

func (j *workerJob) terminate() {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.handle != 0 {
		terminateJobObject.Call(j.handle, 1)
	}
}

func (j *workerJob) close() {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.handle != 0 {
		closeHandle.Call(j.handle)
		j.handle = 0
	}
}
