package library

func (c EffectiveLibraryConfig) ScannerEnabled() bool {
	return c.ScanPolicy.ScannerEnabled
}

func (c EffectiveLibraryConfig) RealtimeRefreshEnabled() bool {
	return c.ScanPolicy.RealtimeMonitorEnabled
}

func (c EffectiveLibraryConfig) ScheduledRefreshEnabled() bool {
	return c.ScanPolicy.ScheduledRefreshEnabled
}

func (c EffectiveLibraryConfig) InventoryProbeBatchEnabled() bool {
	return c.ScanPolicy.InventoryProbeBatchEnabled
}
