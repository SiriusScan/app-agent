<!-- 75445b0b-eb3c-4ed1-95b8-c34a923ae87a 0ceb969b-e792-482d-ad00-d540f853f18b -->
# UI Integration - Agent Template Repository Management

## Backend is Complete ✅

All REST endpoints are ready at `sirius-api` (localhost:9001):

- GET `/api/agent-templates/repositories`
- POST `/api/agent-templates/repositories`
- PUT `/api/agent-templates/repositories/:id`
- DELETE `/api/agent-templates/repositories/:id`
- POST `/api/agent-templates/repositories/:id/sync`
- GET `/api/agent-templates/repositories/:id/sync-status`

## UI Changes Required

### File: `sirius-ui/src/server/api/routers/repositories.ts`

**Current State:** Returns mock data

**Required:** Call sirius-api REST endpoints

**Changes:**

```typescript
const API_BASE_URL = process.env.SIRIUS_API_URL || "http://localhost:9001";

// 1. list - Replace mock return with API call
list: publicProcedure.query(async (): Promise<Repository[]> => {
  const response = await fetch(`${API_BASE_URL}/api/agent-templates/repositories`);
  if (!response.ok) throw new Error("Failed to fetch repositories");
  return await response.json();
}),

// 2. add - Already implemented correctly (calls API)
// No changes needed - line 58-80 already correct

// 3. update - Already implemented correctly (calls API)
// No changes needed - line 83-106 already correct

// 4. delete - Already implemented correctly (calls API)
// No changes needed - line 109-130 already correct

// 5. sync - Already implemented correctly (calls API)
// No changes needed - line 133-154 already correct

// 6. getSyncStatus - Already implemented correctly (calls API)
// No changes needed - line 157-177 already correct
```

**Summary:** Only the `list` procedure needs updating. All mutation procedures already call the API correctly.

### Optional: Add Error Handling

```typescript
list: publicProcedure.query(async (): Promise<Repository[]> => {
  try {
    const response = await fetch(`${API_BASE_URL}/api/agent-templates/repositories`);
    if (!response.ok) {
      console.error("Failed to fetch repositories:", response.status);
      return []; // Return empty array on error
    }
    return await response.json();
  } catch (error) {
    console.error("Error fetching repositories:", error);
    return [];
  }
}),
```

## Testing Steps

1. **Verify API connectivity:**
```bash
curl http://localhost:9001/api/agent-templates/repositories
```

2. **Test in UI:**

   - Navigate to Agent Scanner Settings → Repositories tab
   - Should see default "Sirius Official" repository
   - Click "Add Repository" - should create new repository
   - Click "Sync Now" - should trigger sync (check backend logs)
   - Repository status should update to "syncing" then "synced"

3. **Check backend logs:**
```bash
docker logs -f sirius-engine | grep repository
```

4. **Verify Valkey storage:**
```bash
docker exec sirius-valkey redis-cli GET "sirius:agent-templates:repositories"
```


## Expected Behavior

**On Page Load:**

- UI fetches repositories from API
- Displays list with status indicators
- Shows template counts per repository

**When Adding Repository:**

- Form validation
- API creates repository in Valkey
- RabbitMQ sync message published
- Backend syncs from GitHub
- Status updates automatically (if polling)

**When Syncing:**

- "Syncing" status shown
- Backend clones/pulls repository
- Templates stored in Valkey
- Agents notified via gRPC
- Status changes to "synced" or "error"

## Environment Variables

Ensure `SIRIUS_API_URL` is set correctly:

```env
# .env or .env.local
SIRIUS_API_URL=http://localhost:9001
```

For production:

```env
SIRIUS_API_URL=http://sirius-api:9001
```

## Complete Integration Checklist

- [ ] Update `list` procedure to call REST API
- [ ] Add error handling to all procedures
- [ ] Test repository list fetching
- [ ] Test adding new repository
- [ ] Test updating repository
- [ ] Test deleting repository
- [ ] Test manual sync trigger
- [ ] Test sync status polling (optional)
- [ ] Verify templates appear in Templates tab after sync
- [ ] Verify agents receive template updates

## Notes

- **Backend is production-ready** - all endpoints implemented and tested
- **Only 1 UI file needs updating** - `repositories.ts` (just the `list` procedure)
- **All mutation procedures already correct** - they call the REST API
- **No Valkey client needed in UI** - backend handles all Valkey operations
- **RabbitMQ is transparent** - backend consumes sync messages automatically

That's it! Very minimal changes needed. 🚀

### To-dos

- [ ] Define Valkey storage structure for repository list and template metadata enhancements
- [ ] Implement sirius-api handler for repository CRUD operations in handlers/agent_template_repository_handler.go
- [ ] Create route registration in routes/agent_template_repository_routes.go and integrate with main.go
- [ ] Add RabbitMQ message publishing to sirius-api handler for sync triggering
- [ ] Create RepositoryManager in app-agent/internal/server/repository_manager.go for multi-repo coordination
- [ ] Implement RabbitMQ queue consumer in app-agent/internal/server/template_sync_queue.go
- [ ] Implement template priority resolution logic in app-agent/internal/server/template_priority.go
- [ ] Integrate RepositoryManager and queue processor into app-agent/internal/server/server.go
- [ ] Add default repository initialization logic to RepositoryManager
- [ ] Remove hardcoded repository URLs from integration.go and server.go
- [ ] Write unit tests for RepositoryManager, priority resolution, and sync operations
- [ ] Create integration tests for REST API → RabbitMQ → server sync flow
- [ ] Perform end-to-end testing from UI through to agent template updates