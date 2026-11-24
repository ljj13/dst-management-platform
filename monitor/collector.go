package monitor

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"

	"dst-management-platform-api/logger"
)

type staticInfo struct {
	cpuModel      string
	cpuMhz        float64
	cores         int
	physicalCores int
	hostname      string
	os            string
	platform      string
	platformVer   string
	kernelArch    string
	bootTime      uint64
}

type historyRing struct {
	mu   sync.RWMutex
	buf  []Point
	next int
	full bool
}

func newHistoryRing() *historyRing {
	return &historyRing{buf: make([]Point, MaxHistoryPoints)}
}

func (r *historyRing) append(p Point) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.buf[r.next] = p
	r.next++
	if r.next == len(r.buf) {
		r.next = 0
		r.full = true
	}
}

func (r *historyRing) since(from int64) []Point {
	r.mu.RLock()
	defer r.mu.RUnlock()

	n := r.next
	if r.full {
		n = len(r.buf)
	}
	out := make([]Point, 0, n)
	for i := 0; i < n; i++ {
		idx := i
		if r.full {
			idx = (r.next + i) % len(r.buf)
		}
		if r.buf[idx].Timestamp >= from {
			out = append(out, r.buf[idx])
		}
	}
	return out
}

// Collector 是单例采集器。除了 snapshot/history 以外，所有字段只在采集
// goroutine 中读写。
type Collector struct {
	mu       sync.RWMutex
	snapshot *Snapshot
	history  *historyRing

	startOnce sync.Once
	stopOnce  sync.Once
	stopCh    chan struct{}

	started time.Time
	static  staticInfo

	prevCPU  map[string]cpu.TimesStat
	prevNet  map[string]net.IOCountersStat
	prevDisk map[string]disk.IOCountersStat
	prevProc map[string]float64

	cpuAt  time.Time
	netAt  time.Time
	diskAt time.Time
	procAt time.Time

	warnMu   sync.Mutex
	lastWarn map[string]time.Time
}

// Default 是进程级采集器。所有 HTTP 请求共享它。
var Default = &Collector{
	history:  newHistoryRing(),
	stopCh:   make(chan struct{}),
	prevCPU:  make(map[string]cpu.TimesStat),
	prevNet:  make(map[string]net.IOCountersStat),
	prevDisk: make(map[string]disk.IOCountersStat),
	prevProc: make(map[string]float64),
	lastWarn: make(map[string]time.Time),
}

// Start 启动后台采集。重复调用无副作用。
func Start() { Default.Start() }

// Stop 停止后台采集，仅用于进程退出或测试清理。
func Stop() { Default.Stop() }

// Current 返回最近一次采样；首次采样完成前返回 nil。
func Current() *Snapshot { return Default.Current() }

// History 查询并降采样历史曲线。
func History(rangeName string) (HistoryData, error) { return Default.History(rangeName) }

// QueryProcesses 在最近快照中搜索、排序进程。
func QueryProcesses(sortBy, order, query string, limit int) ProcessQuery {
	return Default.QueryProcesses(sortBy, order, query, limit)
}

func (c *Collector) Start() {
	c.startOnce.Do(func() {
		c.started = time.Now()
		c.static = readStaticInfo()
		c.prime()
		go c.loop()
	})
}

func (c *Collector) Stop() {
	c.stopOnce.Do(func() { close(c.stopCh) })
}

func (c *Collector) loop() {
	ticker := time.NewTicker(IntervalSeconds * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.collect()
		case <-c.stopCh:
			return
		}
	}
}

func (c *Collector) Current() *Snapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.snapshot
}

func (c *Collector) prime() {
	now := time.Now()
	if stats, err := cpu.Times(true); err == nil {
		c.prevCPU = indexCPUTimes(stats)
		c.cpuAt = now
	}
	if counters, err := net.IOCounters(true); err == nil {
		c.prevNet = indexNetCounters(counters)
		c.netAt = now
	}
	if counters, err := disk.IOCounters(); err == nil {
		c.prevDisk = indexDiskCounters(counters)
		c.diskAt = now
	}
	c.procAt = now
}

func (c *Collector) collect() {
	now := time.Now()
	warnings := make([]Warning, 0, 6)
	addWarning := func(source, msg string) {
		for _, w := range warnings {
			if w.Source == source {
				return
			}
		}
		warnings = append(warnings, Warning{Source: source, Message: msg})
		c.logWarn(source, msg)
	}

	cpuStats := c.collectCPU(now, addWarning)
	memoryStats, swapStats := c.collectMemory(addWarning)
	disks := c.collectDisks(addWarning)
	diskIO := c.collectDiskIO(now, addWarning)
	network := c.collectNetwork(now, addWarning)
	processes, processTotal, processPartial := c.collectProcesses(now, memoryStats.Total, addWarning)

	uptime, err := host.Uptime()
	if err != nil {
		uptime = uint64(now.Sub(c.started).Seconds())
	}

	snapshot := &Snapshot{
		Timestamp:       now.Unix(),
		IntervalSeconds: IntervalSeconds,
		Uptime:          uptime,
		Host: HostStats{
			Hostname:        c.static.hostname,
			OS:              c.static.os,
			Platform:        c.static.platform,
			PlatformVersion: c.static.platformVer,
			KernelArch:      c.static.kernelArch,
			BootTime:        c.static.bootTime,
		},
		CPU:            cpuStats,
		Memory:         memoryStats,
		Swap:           swapStats,
		Disks:          disks,
		DiskIO:         diskIO,
		Network:        network,
		Processes:      processes,
		ProcessTotal:   processTotal,
		ProcessPartial: processPartial,
		Warnings:       warnings,
	}

	c.appendHistory(snapshot)

	c.mu.Lock()
	c.snapshot = snapshot
	c.mu.Unlock()
}

func (c *Collector) appendHistory(s *Snapshot) {
	p := Point{
		Timestamp:     s.Timestamp,
		CPU:           s.CPU.Usage,
		PerCore:       append([]float64(nil), s.CPU.PerCore...),
		User:          s.CPU.User,
		System:        s.CPU.System,
		Idle:          s.CPU.Idle,
		Iowait:        s.CPU.Iowait,
		Steal:         s.CPU.Steal,
		Load1:         s.CPU.Load1,
		MemoryUsed:    s.Memory.Used,
		MemoryPercent: s.Memory.UsedPercent,
		SwapUsed:      s.Swap.Used,
		SwapPercent:   s.Swap.UsedPercent,
		DiskReadBps:   s.DiskIO.ReadBytesPerSec,
		DiskWriteBps:  s.DiskIO.WriteBytesPerSec,
		NetUpBps:      s.Network.UpBps,
		NetDownBps:    s.Network.DownBps,
	}
	for _, proc := range s.Processes {
		switch proc.Role {
		case RoleDSTMaster:
			p.MasterCPU += proc.CPU
			p.MasterRSS += proc.RSS
		case RoleDSTCaves:
			p.CavesCPU += proc.CPU
			p.CavesRSS += proc.RSS
		case RoleDMP:
			p.DMPCPU += proc.CPU
			p.DMPRSS += proc.RSS
		}
	}
	c.history.append(p)
}

func (c *Collector) History(rangeName string) (HistoryData, error) {
	if rangeName == "" {
		rangeName = "1h"
	}
	var seconds int64
	var target int
	switch rangeName {
	case "15m":
		seconds, target = 15*60, 180
	case "1h":
		seconds, target = 60*60, 240
	case "24h":
		seconds, target = 24*60*60, 720
	default:
		return HistoryData{}, fmt.Errorf("unsupported range %q", rangeName)
	}

	now := time.Now().Unix()
	from := now - seconds
	points := c.history.since(from)
	downsampled := false
	if len(points) > target {
		points = downsample(points, target)
		downsampled = true
	}

	return HistoryData{
		Range:                rangeName,
		From:                 from,
		To:                   now,
		Available:            len(points) > 0,
		Points:               points,
		Downsampled:          downsampled,
		SampleIntervalSecond: IntervalSeconds,
		RetentionHours:       24,
	}, nil
}

func downsample(points []Point, target int) []Point {
	if target < 2 || len(points) <= target {
		return points
	}
	out := make([]Point, 0, target)
	step := float64(len(points)-1) / float64(target-1)
	last := -1
	for i := 0; i < target; i++ {
		idx := int(math.Round(float64(i) * step))
		if idx == last {
			continue
		}
		if idx >= len(points) {
			idx = len(points) - 1
		}
		out = append(out, points[idx])
		last = idx
	}
	return out
}

func (c *Collector) QueryProcesses(sortBy, order, query string, limit int) ProcessQuery {
	snapshot := c.Current()
	if snapshot == nil {
		return ProcessQuery{Processes: []ProcessStats{}}
	}

	if sortBy != "memory" && sortBy != "name" {
		sortBy = "cpu"
	}
	if order != "asc" {
		order = "desc"
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}

	q := strings.ToLower(strings.TrimSpace(query))
	filtered := make([]ProcessStats, 0, len(snapshot.Processes))
	for _, proc := range snapshot.Processes {
		if q != "" && !processMatches(proc, q) {
			continue
		}
		filtered = append(filtered, proc)
	}
	sortProcesses(filtered, sortBy, order)

	matched := len(filtered)
	if len(filtered) > limit {
		filtered = filtered[:limit]
	}
	return ProcessQuery{
		Timestamp: snapshot.Timestamp,
		Processes: filtered,
		Total:     snapshot.ProcessTotal,
		Matched:   matched,
		Partial:   matched > len(filtered),
	}
}

func processMatches(proc ProcessStats, q string) bool {
	if strings.Contains(strings.ToLower(proc.Name), q) ||
		strings.Contains(strings.ToLower(proc.Role), q) ||
		strings.Contains(strings.ToLower(proc.Shard), q) ||
		strings.Contains(strings.ToLower(proc.Cluster), q) {
		return true
	}
	return strconv.Itoa(int(proc.PID)) == q
}

func sortProcesses(list []ProcessStats, sortBy, order string) {
	sort.SliceStable(list, func(i, j int) bool {
		var less bool
		switch sortBy {
		case "memory":
			less = list[i].RSS < list[j].RSS
		case "name":
			less = strings.ToLower(list[i].Name) < strings.ToLower(list[j].Name)
		default:
			less = list[i].CPU < list[j].CPU
		}
		if order == "asc" {
			return less
		}
		return !less
	})
}

func (c *Collector) collectCPU(now time.Time, addWarning func(string, string)) CPUStats {
	stats, err := cpu.Times(true)
	if err != nil {
		addWarning("cpu", "读取 CPU 计数器失败: "+err.Error())
		return CPUStats{
			Model:         c.static.cpuModel,
			Cores:         c.static.cores,
			PhysicalCores: c.static.physicalCores,
			Mhz:           c.static.cpuMhz,
		}
	}

	elapsed := now.Sub(c.cpuAt).Seconds()
	if elapsed <= 0 || elapsed > 60 {
		elapsed = IntervalSeconds
	}

	result := CPUStats{
		Model:         c.static.cpuModel,
		Cores:         len(stats),
		PhysicalCores: c.static.physicalCores,
		Mhz:           c.static.cpuMhz,
		PerCore:       make([]float64, len(stats)),
	}
	if result.Cores == 0 {
		result.Cores = c.static.cores
	}

	if len(c.prevCPU) > 0 {
		var (
			busySum, userSum, systemSum, idleSum float64
			iowaitSum, stealSum, niceSum         float64
			irqSum, softirqSum                   float64
			measured                             int
		)
		for i, current := range stats {
			key := cpuKey(current, i)
			previous, ok := c.prevCPU[key]
			if !ok {
				continue
			}
			d := diffCPUTimes(current, previous)
			busy := d.total - d.idle - d.iowait
			if busy < 0 {
				busy = 0
			}
			percent := clampPercent(busy / elapsed * 100)
			result.PerCore[i] = round2(percent)

			busySum += busy
			userSum += d.user
			systemSum += d.system
			idleSum += d.idle
			iowaitSum += d.iowait
			stealSum += d.steal
			niceSum += d.nice
			irqSum += d.irq
			softirqSum += d.softirq
			measured++
		}
		if measured > 0 {
			divisor := elapsed * float64(measured)
			result.Usage = round2(clampPercent(busySum / divisor * 100))
			result.User = round2(clampPercent(userSum / divisor * 100))
			result.System = round2(clampPercent(systemSum / divisor * 100))
			result.Idle = round2(clampPercent(idleSum / divisor * 100))
			result.Iowait = round2(clampPercent(iowaitSum / divisor * 100))
			result.Steal = round2(clampPercent(stealSum / divisor * 100))
			result.Nice = round2(clampPercent(niceSum / divisor * 100))
			result.Irq = round2(clampPercent(irqSum / divisor * 100))
			result.Softirq = round2(clampPercent(softirqSum / divisor * 100))
		}
	}

	if avg, err := load.Avg(); err == nil {
		result.Load1 = round2(avg.Load1)
		result.Load5 = round2(avg.Load5)
		result.Load15 = round2(avg.Load15)
	}

	c.prevCPU = indexCPUTimes(stats)
	c.cpuAt = now
	return result
}

type cpuDiff struct {
	user, system, idle, iowait, steal, nice, irq, softirq, total float64
}

func diffCPUTimes(current, previous cpu.TimesStat) cpuDiff {
	d := func(a, b float64) float64 {
		v := a - b
		if v < 0 {
			return 0
		}
		return v
	}
	out := cpuDiff{
		user:    d(current.User, previous.User),
		system:  d(current.System, previous.System),
		idle:    d(current.Idle, previous.Idle),
		iowait:  d(current.Iowait, previous.Iowait),
		steal:   d(current.Steal, previous.Steal),
		nice:    d(current.Nice, previous.Nice),
		irq:     d(current.Irq, previous.Irq),
		softirq: d(current.Softirq, previous.Softirq),
	}
	out.total = out.user + out.system + out.idle + out.iowait + out.steal + out.nice + out.irq + out.softirq
	return out
}

func (c *Collector) collectMemory(addWarning func(string, string)) (MemoryStats, SwapStats) {
	var memory MemoryStats
	if vm, err := mem.VirtualMemory(); err != nil {
		addWarning("memory", "读取内存信息失败: "+err.Error())
	} else {
		memory = MemoryStats{
			Total:       vm.Total,
			Used:        vm.Used,
			Available:   vm.Available,
			Free:        vm.Free,
			Buffers:     vm.Buffers,
			Cached:      vm.Cached,
			BuffCache:   vm.Buffers + vm.Cached,
			UsedPercent: round2(vm.UsedPercent),
		}
	}

	var swap SwapStats
	if sm, err := mem.SwapMemory(); err != nil {
		addWarning("swap", "读取 Swap 信息失败: "+err.Error())
	} else {
		swap = SwapStats{
			Total:       sm.Total,
			Used:        sm.Used,
			Free:        sm.Free,
			UsedPercent: round2(sm.UsedPercent),
			Sin:         sm.Sin,
			Sout:        sm.Sout,
		}
	}
	return memory, swap
}

func (c *Collector) collectDisks(addWarning func(string, string)) []DiskStats {
	parts, err := disk.Partitions(false)
	if err != nil {
		addWarning("disk", "读取文件系统列表失败: "+err.Error())
		return []DiskStats{}
	}

	seen := make(map[string]struct{}, len(parts))
	disks := make([]DiskStats, 0, len(parts))
	for _, part := range parts {
		if _, ok := seen[part.Mountpoint]; ok {
			continue
		}
		seen[part.Mountpoint] = struct{}{}

		usage, err := disk.Usage(part.Mountpoint)
		if err != nil || usage == nil || usage.Total == 0 {
			continue
		}
		fstype := usage.Fstype
		if fstype == "" {
			fstype = part.Fstype
		}
		disks = append(disks, DiskStats{
			Device:            part.Device,
			Mountpoint:        part.Mountpoint,
			Fstype:            fstype,
			Total:             usage.Total,
			Used:              usage.Used,
			Free:              usage.Free,
			UsedPercent:       round2(usage.UsedPercent),
			InodesTotal:       usage.InodesTotal,
			InodesUsed:        usage.InodesUsed,
			InodesFree:        usage.InodesFree,
			InodesUsedPercent: round2(usage.InodesUsedPercent),
		})
	}
	sort.SliceStable(disks, func(i, j int) bool { return disks[i].Mountpoint < disks[j].Mountpoint })
	if len(disks) > 32 {
		disks = disks[:32]
	}
	return disks
}

func (c *Collector) collectDiskIO(now time.Time, addWarning func(string, string)) DiskIOStats {
	counters, err := disk.IOCounters()
	if err != nil {
		addWarning("disk_io", "读取磁盘 I/O 失败: "+err.Error())
		return DiskIOStats{Devices: []DiskDeviceIO{}}
	}

	elapsed := now.Sub(c.diskAt).Seconds()
	if elapsed <= 0 || elapsed > 60 {
		elapsed = IntervalSeconds
	}

	result := DiskIOStats{Devices: make([]DiskDeviceIO, 0, len(counters))}
	keys := make([]string, 0, len(counters))
	for name := range counters {
		keys = append(keys, name)
	}
	sort.Strings(keys)

	for _, name := range keys {
		current := counters[name]
		device := DiskDeviceIO{
			Name:           name,
			ReadBytes:      current.ReadBytes,
			WriteBytes:     current.WriteBytes,
			ReadCount:      current.ReadCount,
			WriteCount:     current.WriteCount,
			IopsInProgress: current.IopsInProgress,
			IoTime:         current.IoTime,
			WeightedIO:     current.WeightedIO,
		}
		if previous, ok := c.prevDisk[name]; ok {
			device.ReadBytesPerSec = rate(current.ReadBytes, previous.ReadBytes, elapsed)
			device.WriteBytesPerSec = rate(current.WriteBytes, previous.WriteBytes, elapsed)
			device.ReadCountPerSec = rate(current.ReadCount, previous.ReadCount, elapsed)
			device.WriteCountPerSec = rate(current.WriteCount, previous.WriteCount, elapsed)
		}
		result.ReadBytes += device.ReadBytes
		result.WriteBytes += device.WriteBytes
		result.ReadBytesPerSec += device.ReadBytesPerSec
		result.WriteBytesPerSec += device.WriteBytesPerSec
		result.ReadCount += device.ReadCount
		result.WriteCount += device.WriteCount
		result.ReadCountPerSec += device.ReadCountPerSec
		result.WriteCountPerSec += device.WriteCountPerSec
		result.IopsInProgress += device.IopsInProgress
		result.IoTime += device.IoTime
		result.WeightedIO += device.WeightedIO
		result.Devices = append(result.Devices, device)
	}

	sort.SliceStable(result.Devices, func(i, j int) bool {
		left := result.Devices[i].ReadBytesPerSec + result.Devices[i].WriteBytesPerSec
		right := result.Devices[j].ReadBytesPerSec + result.Devices[j].WriteBytesPerSec
		return left > right
	})
	if len(result.Devices) > 16 {
		result.Devices = result.Devices[:16]
	}
	result.ReadBytesPerSec = round2(result.ReadBytesPerSec)
	result.WriteBytesPerSec = round2(result.WriteBytesPerSec)
	result.ReadCountPerSec = round2(result.ReadCountPerSec)
	result.WriteCountPerSec = round2(result.WriteCountPerSec)

	c.prevDisk = indexDiskCounters(counters)
	c.diskAt = now
	return result
}

func (c *Collector) collectNetwork(now time.Time, addWarning func(string, string)) NetworkStats {
	counters, err := net.IOCounters(true)
	if err != nil {
		addWarning("network", "读取网卡计数器失败: "+err.Error())
		return NetworkStats{Interfaces: []NetInterface{}}
	}

	elapsed := now.Sub(c.netAt).Seconds()
	if elapsed <= 0 || elapsed > 60 {
		elapsed = IntervalSeconds
	}

	result := NetworkStats{Interfaces: make([]NetInterface, 0, len(counters))}
	for _, current := range counters {
		iface := NetInterface{
			Name:          current.Name,
			BytesSent:     current.BytesSent,
			BytesRecv:     current.BytesRecv,
			PacketsSent:   current.PacketsSent,
			PacketsRecv:   current.PacketsRecv,
			Virtual:       isVirtualInterface(current.Name),
			LinkSpeedMbps: linkSpeedMbps(current.Name),
		}
		if previous, ok := c.prevNet[current.Name]; ok {
			iface.UpBps = rate(current.BytesSent, previous.BytesSent, elapsed)
			iface.DownBps = rate(current.BytesRecv, previous.BytesRecv, elapsed)
			iface.UpMbps = round2(iface.UpBps * 8 / 1_000_000)
			iface.DownMbps = round2(iface.DownBps * 8 / 1_000_000)
			iface.UpMBps = round2(iface.UpBps / 1024 / 1024)
			iface.DownMBps = round2(iface.DownBps / 1024 / 1024)
		}
		if !iface.Virtual {
			result.UpBps += iface.UpBps
			result.DownBps += iface.DownBps
		}
		result.Interfaces = append(result.Interfaces, iface)
	}

	sort.SliceStable(result.Interfaces, func(i, j int) bool {
		return result.Interfaces[i].Name < result.Interfaces[j].Name
	})
	result.UpBps = round2(result.UpBps)
	result.DownBps = round2(result.DownBps)
	result.UpMbps = round2(result.UpBps * 8 / 1_000_000)
	result.DownMbps = round2(result.DownBps * 8 / 1_000_000)
	result.UpMBps = round2(result.UpBps / 1024 / 1024)
	result.DownMBps = round2(result.DownBps / 1024 / 1024)

	c.prevNet = indexNetCounters(counters)
	c.netAt = now
	return result
}

func readStaticInfo() staticInfo {
	info := staticInfo{}
	if list, err := cpu.Info(); err == nil && len(list) > 0 {
		info.cpuModel = strings.TrimSpace(list[0].ModelName)
		info.cpuMhz = round2(list[0].Mhz)
	}
	if info.cpuModel == "" {
		info.cpuModel = "unknown"
	}
	if n, err := cpu.Counts(true); err == nil {
		info.cores = n
	}
	if n, err := cpu.Counts(false); err == nil {
		info.physicalCores = n
	}
	if n, err := host.Info(); err == nil {
		info.hostname = n.Hostname
		info.os = n.OS
		info.platform = n.Platform
		info.platformVer = n.PlatformVersion
		info.kernelArch = n.KernelArch
		info.bootTime = n.BootTime
	}
	return info
}

func indexCPUTimes(stats []cpu.TimesStat) map[string]cpu.TimesStat {
	out := make(map[string]cpu.TimesStat, len(stats))
	for i, stat := range stats {
		out[cpuKey(stat, i)] = stat
	}
	return out
}

func cpuKey(stat cpu.TimesStat, index int) string {
	if stat.CPU != "" {
		return stat.CPU
	}
	return strconv.Itoa(index)
}

func indexNetCounters(counters []net.IOCountersStat) map[string]net.IOCountersStat {
	out := make(map[string]net.IOCountersStat, len(counters))
	for _, counter := range counters {
		out[counter.Name] = counter
	}
	return out
}

func indexDiskCounters(counters map[string]disk.IOCountersStat) map[string]disk.IOCountersStat {
	out := make(map[string]disk.IOCountersStat, len(counters))
	for name, counter := range counters {
		out[name] = counter
	}
	return out
}

func rate(current, previous uint64, elapsed float64) float64 {
	if current < previous || elapsed <= 0 {
		return 0
	}
	return round2(float64(current-previous) / elapsed)
}

func linkSpeedMbps(name string) int64 {
	if name == "" || strings.ContainsAny(name, `/\`) {
		return 0
	}
	data, err := os.ReadFile(filepath.Join("/sys/class/net", name, "speed"))
	if err != nil {
		return 0
	}
	speed, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil || speed < 0 {
		return 0
	}
	return speed
}

func isVirtualInterface(name string) bool {
	lower := strings.ToLower(name)
	if lower == "lo" || lower == "lo0" {
		return true
	}
	for _, prefix := range []string{"docker", "veth", "br-", "virbr", "tun", "tap", "wg", "cni", "flannel", "cali", "tailscale"} {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}

func clampPercent(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func (c *Collector) logWarn(source, message string) {
	c.warnMu.Lock()
	last, ok := c.lastWarn[source]
	if !ok || time.Since(last) > 5*time.Minute {
		c.lastWarn[source] = time.Now()
		c.warnMu.Unlock()
		if logger.Logger != nil {
			logger.Logger.Warnf("[monitor] %s: %s", source, message)
		}
		return
	}
	c.warnMu.Unlock()
}

// readProcessList 单独拆分，便于进程查询逻辑复用。
func readProcessList() ([]*process.Process, error) {
	return process.Processes()
}
