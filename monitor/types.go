// Package monitor 采集主机与 DST/DMP 进程的只读运行指标。
//
// 采集器是进程级单例：每 5 秒采样一次，当前快照与历史环形缓冲区都只保留
// 一份，多个浏览器连接共享同一份数据，不会为每个连接重复采样。
package monitor

// IntervalSeconds 是采集周期。前端与 API 都使用这个值，避免两处写死。
const IntervalSeconds = 5

// RetentionSeconds 是历史环形缓冲区的覆盖时长（24 小时）。
const RetentionSeconds = 24 * 60 * 60

// MaxHistoryPoints 是 24 小时 / 5 秒的采样点数上限。
const MaxHistoryPoints = RetentionSeconds / IntervalSeconds

// maxProcesses 是当前快照中最多保留的进程数。进程表查询与 CPU/内存曲线
// 共用这份数据，避免每个请求重新枚举进程。
const maxProcesses = 300

// Role 是进程在 DMP 视角下的角色。
const (
	RoleDMP       = "dmp"
	RoleDSTMaster = "dst-master"
	RoleDSTCaves  = "dst-caves"
	RoleDST       = "dst"
	RoleOther     = "other"
)

// Warning 描述某一类指标采集失败的原因。API 原样返回，前端显示真实状态，
// 不用 0 或假数据掩盖失败。
type Warning struct {
	Source  string `json:"source"`
	Message string `json:"message"`
}

// HostStats 是变化很慢的主机静态信息。
type HostStats struct {
	Hostname        string `json:"hostname"`
	OS              string `json:"os"`
	Platform        string `json:"platform"`
	PlatformVersion string `json:"platformVersion"`
	KernelArch      string `json:"kernelArch"`
	BootTime        uint64 `json:"bootTime"`
}

// CPUStats 是整机与每个逻辑 CPU 的实时指标。
type CPUStats struct {
	Usage         float64   `json:"usage"`
	PerCore       []float64 `json:"perCore"`
	Model         string    `json:"model"`
	Cores         int       `json:"cores"`
	PhysicalCores int       `json:"physicalCores"`
	Mhz           float64   `json:"mhz"`
	Load1         float64   `json:"load1"`
	Load5         float64   `json:"load5"`
	Load15        float64   `json:"load15"`
	User          float64   `json:"user"`
	System        float64   `json:"system"`
	Idle          float64   `json:"idle"`
	Iowait        float64   `json:"iowait"`
	Steal         float64   `json:"steal"`
	Nice          float64   `json:"nice"`
	Irq           float64   `json:"irq"`
	Softirq       float64   `json:"softirq"`
}

// MemoryStats 是物理内存指标，单位均为字节。
type MemoryStats struct {
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Available   uint64  `json:"available"`
	Free        uint64  `json:"free"`
	Buffers     uint64  `json:"buffers"`
	Cached      uint64  `json:"cached"`
	BuffCache   uint64  `json:"buffCache"`
	UsedPercent float64 `json:"usedPercent"`
}

// SwapStats 是交换分区指标，单位均为字节。
type SwapStats struct {
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	UsedPercent float64 `json:"usedPercent"`
	Sin         uint64  `json:"sin"`
	Sout        uint64  `json:"sout"`
}

// DiskStats 是文件系统空间指标。
type DiskStats struct {
	Device            string  `json:"device"`
	Mountpoint        string  `json:"mountpoint"`
	Fstype            string  `json:"fstype"`
	Total             uint64  `json:"total"`
	Used              uint64  `json:"used"`
	Free              uint64  `json:"free"`
	UsedPercent       float64 `json:"usedPercent"`
	InodesTotal       uint64  `json:"inodesTotal"`
	InodesUsed        uint64  `json:"inodesUsed"`
	InodesFree        uint64  `json:"inodesFree"`
	InodesUsedPercent float64 `json:"inodesUsedPercent"`
}

// DiskDeviceIO 是单块磁盘的累计量与速率。
type DiskDeviceIO struct {
	Name             string  `json:"name"`
	ReadBytes        uint64  `json:"readBytes"`
	WriteBytes       uint64  `json:"writeBytes"`
	ReadBytesPerSec  float64 `json:"readBytesPerSec"`
	WriteBytesPerSec float64 `json:"writeBytesPerSec"`
	ReadCount        uint64  `json:"readCount"`
	WriteCount       uint64  `json:"writeCount"`
	ReadCountPerSec  float64 `json:"readCountPerSec"`
	WriteCountPerSec float64 `json:"writeCountPerSec"`
	IopsInProgress   uint64  `json:"iopsInProgress"`
	IoTime           uint64  `json:"ioTime"`
	WeightedIO       uint64  `json:"weightedIo"`
}

// DiskIOStats 是所有磁盘聚合后的 I/O 指标。
type DiskIOStats struct {
	ReadBytes        uint64         `json:"readBytes"`
	WriteBytes       uint64         `json:"writeBytes"`
	ReadBytesPerSec  float64        `json:"readBytesPerSec"`
	WriteBytesPerSec float64        `json:"writeBytesPerSec"`
	ReadCount        uint64         `json:"readCount"`
	WriteCount       uint64         `json:"writeCount"`
	ReadCountPerSec  float64        `json:"readCountPerSec"`
	WriteCountPerSec float64        `json:"writeCountPerSec"`
	IopsInProgress   uint64         `json:"iopsInProgress"`
	IoTime           uint64         `json:"ioTime"`
	WeightedIO       uint64         `json:"weightedIo"`
	Devices          []DiskDeviceIO `json:"devices"`
}

// NetInterface 是单块网卡的速率与累计量。up/down 同时给出 B/s、Mbps 和
// MB/s，前端必须按标签区分比特与字节，不能把 Mbps 当成 MB/s。
type NetInterface struct {
	Name          string  `json:"name"`
	UpBps         float64 `json:"upBps"`
	DownBps       float64 `json:"downBps"`
	UpMbps        float64 `json:"upMbps"`
	DownMbps      float64 `json:"downMbps"`
	UpMBps        float64 `json:"upMBps"`
	DownMBps      float64 `json:"downMBps"`
	BytesSent     uint64  `json:"bytesSent"`
	BytesRecv     uint64  `json:"bytesRecv"`
	PacketsSent   uint64  `json:"packetsSent"`
	PacketsRecv   uint64  `json:"packetsRecv"`
	LinkSpeedMbps int64   `json:"linkSpeedMbps"`
	Virtual       bool    `json:"virtual"`
}

// NetworkStats 汇总非回环、非虚拟网卡的流量。
type NetworkStats struct {
	UpBps      float64        `json:"upBps"`
	DownBps    float64        `json:"downBps"`
	UpMbps     float64        `json:"upMbps"`
	DownMbps   float64        `json:"downMbps"`
	UpMBps     float64        `json:"upMBps"`
	DownMBps   float64        `json:"downMBps"`
	Interfaces []NetInterface `json:"interfaces"`
}

// ProcessStats 是单个进程的只读指标。绝不返回命令行、环境变量或凭据。
type ProcessStats struct {
	PID           int32   `json:"pid"`
	Name          string  `json:"name"`
	CPU           float64 `json:"cpu"`
	RSS           uint64  `json:"rss"`
	MemoryPercent float64 `json:"memoryPercent"`
	Status        string  `json:"status"`
	Uptime        int64   `json:"uptime"`
	CreateTime    int64   `json:"createTime"`
	Role          string  `json:"role"`
	Shard         string  `json:"shard,omitempty"`
	Cluster       string  `json:"cluster,omitempty"`
}

// Snapshot 是一次采样得到的完整只读快照。
type Snapshot struct {
	Timestamp       int64          `json:"timestamp"`
	IntervalSeconds int            `json:"intervalSeconds"`
	Uptime          uint64         `json:"uptime"`
	Host            HostStats      `json:"host"`
	CPU             CPUStats       `json:"cpu"`
	Memory          MemoryStats    `json:"memory"`
	Swap            SwapStats      `json:"swap"`
	Disks           []DiskStats    `json:"disks"`
	DiskIO          DiskIOStats    `json:"diskIo"`
	Network         NetworkStats   `json:"network"`
	Processes       []ProcessStats `json:"processes"`
	ProcessTotal    int            `json:"processTotal"`
	ProcessPartial  bool           `json:"processPartial"`
	Warnings        []Warning      `json:"warnings"`
}

// Point 是历史曲线点。只保留绘图需要的字段，24 小时环形缓冲不会因为
// 进程明细而无限增长。
type Point struct {
	Timestamp     int64     `json:"timestamp"`
	CPU           float64   `json:"cpu"`
	PerCore       []float64 `json:"perCore"`
	User          float64   `json:"user"`
	System        float64   `json:"system"`
	Idle          float64   `json:"idle"`
	Iowait        float64   `json:"iowait"`
	Steal         float64   `json:"steal"`
	Load1         float64   `json:"load1"`
	MemoryUsed    uint64    `json:"memoryUsed"`
	MemoryPercent float64   `json:"memoryPercent"`
	SwapUsed      uint64    `json:"swapUsed"`
	SwapPercent   float64   `json:"swapPercent"`
	DiskReadBps   float64   `json:"diskReadBps"`
	DiskWriteBps  float64   `json:"diskWriteBps"`
	NetUpBps      float64   `json:"netUpBps"`
	NetDownBps    float64   `json:"netDownBps"`
	MasterCPU     float64   `json:"masterCpu"`
	MasterRSS     uint64    `json:"masterRss"`
	CavesCPU      float64   `json:"cavesCpu"`
	CavesRSS      uint64    `json:"cavesRss"`
	DMPCPU        float64   `json:"dmpCpu"`
	DMPRSS        uint64    `json:"dmpRss"`
}

// HistoryData 是历史查询响应。
type HistoryData struct {
	Range                string  `json:"range"`
	From                 int64   `json:"from"`
	To                   int64   `json:"to"`
	Available            bool    `json:"available"`
	Points               []Point `json:"points"`
	Downsampled          bool    `json:"downsampled"`
	SampleIntervalSecond int     `json:"sampleIntervalSeconds"`
	RetentionHours       int     `json:"retentionHours"`
}

// ProcessQuery 是进程表查询响应。
type ProcessQuery struct {
	Timestamp int64          `json:"timestamp"`
	Processes []ProcessStats `json:"processes"`
	Total     int            `json:"total"`
	Matched   int            `json:"matched"`
	Partial   bool           `json:"partial"`
}
