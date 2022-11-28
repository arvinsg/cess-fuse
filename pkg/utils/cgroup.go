package utils

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	CGroupPath         = "/proc/self/cgroup"
	CGroupFolderPrefix = "/sys/fs/cgroup/memory"
	MemLimitFileSuffix = "/memory.limit_in_bytes"
	MemUsageFileSuffix = "/memory.usage_in_bytes"
)

func GetCGroupAvailableMem() (retVal uint64, err error) {
	//get the memory cgroup for self and send limit - usage for the cgroup
	data, err := ioutil.ReadFile(CGroupPath)
	if err != nil {
		return 0, err
	}

	path, err := getMemoryCGroupPath(string(data))
	if err != nil {
		return 0, err
	}

	// newer version of docker mounts the cgroup memory limit/usage files directly under
	// /sys/fs/cgroup/memory/ rather than /sys/fs/cgroup/memory/docker/$container_id/
	if _, err := os.Stat(filepath.Join(CGroupFolderPrefix, path)); os.IsExist(err) {
		path = filepath.Join(CGroupFolderPrefix, path)
	} else {
		path = filepath.Join(CGroupFolderPrefix)
	}

	memLimit, err := readFileAndGetValue(filepath.Join(path, MemLimitFileSuffix))
	if err != nil {
		return 0, err
	}

	memUsage, err := readFileAndGetValue(filepath.Join(path, MemUsageFileSuffix))
	if err != nil {
		return 0, err
	}

	return memLimit - memUsage, nil
}

func getMemoryCGroupPath(data string) (string, error) {
	/*
	   Content of /proc/self/cgroup

	   11:hugetlb:/
	   10:memory:/user.slice
	   9:cpuset:/
	   8:blkio:/user.slice
	   7:perf_event:/
	   6:net_prio,net_cls:/
	   5:cpuacct,cpu:/user.slice
	   4:devices:/user.slice
	   3:freezer:/
	   2:pids:/
	   1:name=systemd:/user.slice/user-1000.slice/session-1759.scope
	*/

	dataArray := strings.Split(data, "\n")
	for index := range dataArray {
		kvArray := strings.Split(dataArray[index], ":")
		if len(kvArray) == 3 {
			if kvArray[1] == "memory" {
				return kvArray[2], nil
			}
		}
	}

	return "", errors.New("unable to get memory cgroup path")
}

func readFileAndGetValue(path string) (uint64, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return 0, err
	}

	return strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
}
