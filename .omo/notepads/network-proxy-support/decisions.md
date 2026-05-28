# Network Proxy Support - Decisions

## 2026-05-28 Decision: Per-channel XFF switch (not global)
- User requested: "前端单独设置XFF 开关来单渠道的决定是否开启XFF header转发"
- Decision: Channel-level XFF switch only, no global default
- ForwardXForwardedFor moved from network_setting to dto.ChannelSettings
- Each channel must explicitly enable XFF forwarding
- Existing channels without the field default to false (omitempty)
- This means XFF is disabled by default for all channels — must be opted in per-channel
