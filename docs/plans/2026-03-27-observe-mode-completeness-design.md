# Observe Mode Completeness -- Design

**Issues:** #76, #77, #78 (partial)
**Date:** 2026-03-27

**Note:** #79 and #80 were already resolved (fields are populated). Closed.
**Note:** #78 BAV field requires SDK update (jacaudi/nextdns-go#37). Only `logs.location` is in scope.

## #76: Setup Section

Add `ObservedSetup` type capturing the read-only setup data from the API:
- `IPv4 []string` -- DNS IPv4 addresses
- `IPv6 []string` -- DNS IPv6 addresses
- `LinkedIP *ObservedLinkedIP` -- Linked IP config (excluding `updateToken` for security)
- `DNSCrypt string` -- DNSCrypt stamp

New types needed:
- `ObservedSetup` struct
- `ObservedLinkedIP` struct (Servers []string, IP string, DDNS string -- no updateToken)

Changes:
- Add `ObservedSetup` to `nextdnsprofile_observed_types.go`
- Add `Setup *ObservedSetup` to `ObservedConfig`
- Add `GetSetup` to `ClientInterface`
- Add `GetSetup` implementation in `client.go`
- Add `GetSetup` to mock client
- Populate in `readFullProfile`
- No suggestedSpec entry (setup is read-only, not user-configurable via spec)

## #77: Parental Control Gaps

Services are already populated. Three fields missing:

### BlockBypass (bool)
- Add `BlockBypass bool` to `ObservedParentalControl`
- Add `BlockBypass *bool` to `ParentalControlSpec`
- Read from `GetParentalControl` in `readFullProfile`
- Wire into managed mode sync in `syncWithNextDNS`
- Pass through in `buildSuggestedSpec`

### Recreation (schedule object)
- Add `ObservedRecreation` type with `Times` and `Timezone` fields
- Add to `ObservedParentalControl`
- Read from `GetParentalControl` in `readFullProfile`
- Observe-only (too complex for spec, recreation schedules aren't user-configurable via the operator)

### Category Recreation (bool per category)
- Add `Recreation bool` to `ObservedCategoryEntry`
- Add `Recreation *bool` to `CategoryEntry`
- Read from `GetParentalControlCategories` in `readFullProfile`
- Pass through in `buildSuggestedSpec`

## #78: Settings Gaps (logs.location only, BAV deferred)

### Logs Location
- Add `Location string` to `ObservedLogs`
- Add `Location string` to `LogsSpec`
- Read from `settings.Logs.Location` in `readFullProfile`
- Pass through in `buildSuggestedSpec`
- Wire into managed mode `UpdateSettings`

### BAV
- Deferred: SDK doesn't have the field (jacaudi/nextdns-go#37)

## Documentation

Update `docs/README.md`:
- Add setup section to observe mode docs
- Add `blockBypass` to parental control spec table
- Add `location` to logs spec table
- Add `recreation` to category entry table
- Update status fields table with new observed fields
