# Testing Guide: Software Inventory Reporting

## Quick Test (macOS)

### 1. Run Template Scan via Server

**Start the agent (if not already running):**
```bash
# From app-agent directory
./bin/sirius-agent server
```

**From server console, send:**
```
internal:template-scan --all
```

### 2. Expected Agent Logs

Look for these log lines in the agent output:

```
DEBUG Collecting software packages for enhanced reporting
DEBUG Collected software packages {"package_count": 42}
DEBUG Converted template results to vulnerabilities {"vulnerability_count": 3}
DEBUG Built software inventory {"package_count": 42}
INFO  Successfully submitted template results to API {
  "vulnerabilities": 3,
  "packages": 42,
  "enhanced_data": true,
  "submission_time": "125ms",
  "host_id": "your-host-id",
  "agent_id": "your-agent-id"
}
```

**Key Indicators:**
- ✅ `"packages": 42` (or any number > 0)
- ✅ `"enhanced_data": true`
- ✅ No API submission errors

### 3. Verify in Database

**Connect to PostgreSQL:**
```bash
# From Sirius directory
docker exec -it sirius-postgres psql -U sirius -d sirius
```

**Query software inventory:**
```sql
-- Check if software_inventory JSONB field is populated
SELECT 
  id,
  ip,
  hostname,
  software_inventory->>'package_count' as pkg_count,
  jsonb_array_length(software_inventory->'packages') as actual_packages,
  software_inventory->>'collected_at' as collected_at,
  software_inventory->>'source' as source
FROM hosts 
WHERE agent_id = 'your-agent-id'  -- Replace with actual agent ID
ORDER BY updated_at DESC
LIMIT 1;
```

**Expected output:**
```
 id | ip          | hostname | pkg_count | actual_packages | collected_at              | source       
----+-------------+----------+-----------+-----------------+---------------------------+--------------
 1  | 192.168.1.5 | macbook  | 42        | 42              | 2025-01-15T12:34:56Z     | sirius-agent
```

**View actual packages:**
```sql
-- See the full package list (formatted)
SELECT jsonb_pretty(software_inventory->'packages')
FROM hosts 
WHERE agent_id = 'your-agent-id'
LIMIT 1;
```

**Expected output:**
```json
[
  {
    "name": "git",
    "source": "homebrew",
    "version": "2.43.0"
  },
  {
    "name": "python3",
    "source": "homebrew",
    "version": "3.12.1"
  },
  ...
]
```

### 4. Verify in UI

**Navigate to:**
1. Open Sirius UI: `http://localhost:3000`
2. Go to "Hosts" page
3. Find your host (by IP or hostname)
4. Click on the host to view details
5. Look for "Software Inventory" or "Packages" tab

**Expected:**
- List of installed packages with names and versions
- Package count matches database
- Last updated timestamp

## Debugging

### If No Packages Collected

**Check agent logs for:**
```
DEBUG Collecting software packages for enhanced reporting
```

**If you DON'T see this line:**
- Check if `agentInfo.ScriptingEnabled` is true (for Windows)
- Check if package gathering failed silently

**To debug, check scan package logs:**
```go
// The GatherMacOSPackages function should log:
agentInfo.Logger.Debug("Gathering macOS packages via Homebrew and system")
```

### If Packages Collected But Not in Database

**Check agent logs for API errors:**
```
ERROR Failed to submit template results to API {"error": "..."}
```

**Common issues:**
1. **Server not reachable:** Check `ApiBaseURL` in agent config
2. **Server endpoint wrong:** Verify server is running and `/host/with-source` endpoint exists
3. **Database connection:** Check server can connect to PostgreSQL

**Test the API directly:**
```bash
curl -X POST http://localhost:8080/host/with-source \
  -H "Content-Type: application/json" \
  -d '{
    "host": {
      "hid": "test-host-1",
      "ip": "192.168.1.99",
      "hostname": "test-machine",
      "os": "macOS",
      "osversion": "14.1.1"
    },
    "source": {
      "name": "sirius-agent",
      "version": "1.0.0-template-mvp",
      "config": "template-scan"
    },
    "software_inventory": {
      "packages": [
        {"name": "test-package", "version": "1.0", "source": "test"}
      ],
      "package_count": 1,
      "collected_at": "2025-01-15T12:00:00Z",
      "source": "sirius-agent"
    }
  }'
```

**Expected response:**
```json
{
  "message": "Host added successfully with enhanced SBOM data",
  "source": "sirius-agent",
  "host_ip": "192.168.1.99",
  "enhanced_data_included": true,
  "software_inventory": true,
  "system_fingerprint": false,
  "agent_metadata": false
}
```

### If Database Shows NULL for software_inventory

**Possible causes:**
1. **API endpoint not implemented:** Check server code has `AddHostWithSourceAndJSONB` function
2. **Wrong API version:** Ensure server is running latest code
3. **Database schema missing:** Check `software_inventory` JSONB column exists

**Verify schema:**
```sql
\d+ hosts
```

Look for:
```
 software_inventory | jsonb | 
 system_fingerprint | jsonb |
 agent_metadata     | jsonb |
```

## Platform-Specific Testing

### macOS (Current Platform)
```bash
# Should collect Homebrew + system packages
./bin/sirius-agent template run-all

# Check logs for:
# - "Gathering macOS packages via Homebrew and system"
# - Package count > 0
```

### Linux
```bash
# Should collect dpkg/rpm packages
./bin/sirius-agent template run-all

# Check logs for:
# - "Gathering Linux packages via dpkg/rpm"
# - Package count > 0
```

### Windows
```bash
# Requires PowerShell scripting enabled
.\bin\sirius-agent.exe template run-all

# Check logs for:
# - "Gathering Windows packages via PowerShell"
# - Package count > 0
```

## Success Criteria

### ✅ Agent Side
- [ ] Packages collected (log shows count > 0)
- [ ] Software inventory formatted (log shows "Built software inventory")
- [ ] API submission successful (log shows "Successfully submitted...")
- [ ] Enhanced data flag is true (log shows `"enhanced_data": true`)

### ✅ Server Side
- [ ] API request received (server logs show POST /host/with-source)
- [ ] Enhanced data processed (server logs show "enhanced_data_included: true")
- [ ] No errors in server logs

### ✅ Database Side
- [ ] Host record exists
- [ ] `software_inventory` JSONB field is NOT NULL
- [ ] Package count matches agent log
- [ ] Package array has correct number of items
- [ ] Each package has name, version, source fields

### ✅ UI Side
- [ ] Host appears in hosts list
- [ ] Host detail page loads
- [ ] Software inventory section visible
- [ ] Packages listed with names and versions
- [ ] Package count displayed

## Troubleshooting Reference

| Symptom | Likely Cause | Fix |
|---------|--------------|-----|
| Log shows 0 packages | Package gathering failed | Check platform-specific commands work manually |
| Log shows packages but DB is NULL | API submission failed | Check ApiBaseURL config, verify server running |
| DB shows count but no array | Wrong API version | Update server to use AddHostWithSourceAndJSONB |
| UI doesn't show packages | Frontend query wrong | Check UI is querying software_inventory field |
| Packages collected but count is 0 | Empty array issue | Check package gathering logic for filters |

## Next Steps After Verification

If everything works:
1. ✅ Mark Task 7.7.1.4 (Integration Testing) as complete
2. ✅ Mark Phase 7.7.1 (Basic REST API Reporting) as complete
3. Consider Phase 7.7.2 (Enhanced Reporting):
   - System fingerprinting (CPU, memory, disk)
   - Enhanced package metadata
   - Incremental updates

If something doesn't work:
1. 🔍 Check logs at each layer (agent → server → database)
2. 🔍 Use debugging commands above
3. 🔍 Verify API endpoints and database schema
4. 🐛 Report issue with specific error messages and logs

---

*Testing Guide Generated: 2025-01-15*  
*Phase: 7.7.1.3 Software Inventory Fix*

