package monitor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
)

func (c *Collector) collectProcesses(
	now time.Time,
	totalMemory uint64,
	addWarning func(string, string),
) ([]ProcessStats, int, bool) {
	procs, err := readProcessList()
	if err != nil {
		addWarning("process", "读取进程列表失败: "+err.Error())
		return []ProcessStats{}, 0, false
	}

	elapsed := now.Sub(c.procAt).Seconds()
	if elapsed <= 0 || elapsed > 60 {
		elapsed = IntervalSeconds
	}

	self := int32(os.Getpid())
	all := make([]ProcessStats, 0, len(procs))
	nextPrev := make(map[string]float64, len(procs))

	for _, p := range procs {
		pid := p.Pid
		name, _ := p.Name()
		exe, _ := p.Exe()
		base := filepath.Base(exe)
		if base == "." || base == string(filepath.Separator) {
			base = ""
		}

		displayName := strings.TrimSpace(name)
		if displayName == "" {
			displayName = strings.TrimSpace(base)
		}
		if displayName == "" {
			displayName = "unknown"
		}

		role := RoleOther
		shard := ""
		cluster := ""
		lowerBase := strings.ToLower(base)
		lowerName := strings.ToLower(name)
		isDST := strings.HasPrefix(lowerBase, "dontstarve_dedicated_server_nullrenderer") ||
			strings.HasPrefix(lowerName, "dontstarve_dedicated_server") ||
			strings.HasPrefix(lowerName, "dontstarve_dedi")

		if pid == self {
			role = RoleDMP
		} else if isDST {
			cmdline, cmdErr := p.Cmdline()
			if cmdErr != nil {
				role = RoleDST
			} else {
				role, shard, cluster = classifyDST(cmdline)
			}
		}

		createTime, _ := p.CreateTime()
		key := fmt.Sprintf("%d:%d", pid, createTime)

		var cpuPercent float64
		if times, timesErr := p.Times(); timesErr == nil && times != nil {
			total := processCPUTotal(*times)
			if previous, ok := c.prevProc[key]; ok && elapsed > 0 && total >= previous {
				cpuPercent = (total - previous) / elapsed * 100
			}
			nextPrev[key] = total
		}

		var rss uint64
		if memory, memErr := p.MemoryInfo(); memErr == nil && memory != nil {
			rss = memory.RSS
		}
		memoryPercent := 0.0
		if totalMemory > 0 {
			memoryPercent = float64(rss) / float64(totalMemory) * 100
		}

		statusText := "unknown"
		if status, statusErr := p.Status(); statusErr == nil && len(status) > 0 {
			statusText = strings.Join(status, ",")
		}

		uptime := int64(0)
		if createTime > 0 {
			created := time.UnixMilli(createTime)
			if now.After(created) {
				uptime = int64(now.Sub(created).Seconds())
			}
		}

		all = append(all, ProcessStats{
			PID:           pid,
			Name:          displayName,
			CPU:           round2(cpuPercent),
			RSS:           rss,
			MemoryPercent: round2(memoryPercent),
			Status:        statusText,
			Uptime:        uptime,
			CreateTime:    createTime,
			Role:          role,
			Shard:         shard,
			Cluster:       cluster,
		})
	}

	c.prevProc = nextPrev
	c.procAt = now

	selected := selectProcesses(all)
	return selected, len(all), len(all) > len(selected)
}

// selectProcesses 保留 CPU 最高的 200 个进程、内存最高的补充进程，以及
// 始终需要展示的 DST/DMP 进程；最多 300 个，避免每 5 秒返回整张进程表。
func selectProcesses(all []ProcessStats) []ProcessStats {
	if len(all) <= maxProcesses {
		out := append([]ProcessStats(nil), all...)
		sortProcesses(out, "cpu", "desc")
		return out
	}

	byCPU := append([]ProcessStats(nil), all...)
	sortProcesses(byCPU, "cpu", "desc")
	byMemory := append([]ProcessStats(nil), all...)
	sortProcesses(byMemory, "memory", "desc")

	seen := make(map[int32]struct{}, maxProcesses)
	selected := make([]ProcessStats, 0, maxProcesses)
	add := func(proc ProcessStats) {
		if len(selected) >= maxProcesses {
			return
		}
		if _, ok := seen[proc.PID]; ok {
			return
		}
		seen[proc.PID] = struct{}{}
		selected = append(selected, proc)
	}

	for i := 0; i < len(byCPU) && i < 200; i++ {
		add(byCPU[i])
	}
	for i := 0; i < len(byMemory) && len(selected) < maxProcesses; i++ {
		add(byMemory[i])
	}
	for _, proc := range all {
		if proc.Role != RoleOther {
			add(proc)
		}
	}
	sortProcesses(selected, "cpu", "desc")
	return selected
}

func processCPUTotal(t cpu.TimesStat) float64 {
	return t.User + t.System + t.Nice + t.Iowait + t.Irq + t.Softirq + t.Steal
}

func classifyDST(cmdline string) (role, shard, cluster string) {
	fields := strings.Fields(cmdline)
	for i := 0; i < len(fields)-1; i++ {
		switch strings.ToLower(fields[i]) {
		case "-shard":
			shard = strings.TrimSpace(fields[i+1])
		case "-cluster":
			cluster = strings.TrimSpace(fields[i+1])
		}
	}
	if len(cluster) > 64 {
		cluster = cluster[:64]
	}
	switch strings.ToLower(shard) {
	case "master":
		return RoleDSTMaster, titleWord(shard), cluster
	case "caves":
		return RoleDSTCaves, titleWord(shard), cluster
	default:
		return RoleDST, titleWord(shard), cluster
	}
}

func titleWord(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	lower := strings.ToLower(value)
	return strings.ToUpper(lower[:1]) + lower[1:]
}
