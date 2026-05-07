package library

type contentShapeCounters struct {
	TokenProfileParses     int
	DirectoryProfileBuilds int
	PlanCompiles           int
	PlanReuses             int
	MaterializationBatches int
}

func (c *contentShapeCounters) snapshot() contentShapeCounters {
	if c == nil {
		return contentShapeCounters{}
	}
	return *c
}
