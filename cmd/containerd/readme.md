plugin.TransferPlugin       -> plugin.GRPCPlugin
plugin.EventPlugin       -> plugin.GRPCPlugin
plugin.EventPlugin       -> plugin.ServicePlugin
plugin.MetadataPlugin       -> plugin.ServicePlugin
plugin.StreamingPlugin      -> plugin.GRPCPlugin
plugin.MetadataPlugin   ->      plugin.LeasePlugin          -> plugin.GRPCPlugin
plugin.GCPlugin            ->      plugin.LeasePlugin          -> plugin.GRPCPlugin
plugin.WarningPlugin      -> plugin.ServicePlugin