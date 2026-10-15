# B-Deploy API Limitations

## Device API v2 - setupId Field Not Returned

### Issue
The B-Deploy Device API v2 (`/rest-device/v2/device/`) does **NOT return the `setupId` field** in GET responses, even though it accepts it in PUT/POST requests.

### Evidence

**PUT Request (Association):**
```bash
PUT /rest-device/v2/device/?_id=689fcb2d03af0e3de789c742
Content-Type: application/json

{
  "_id": "689fcb2d03af0e3de789c742",
  "serial": "UTD416000978",
  "name": "UTD416000978",
  "NetworkName": "gch-control-only",
  "setupId": "6917e8cf04cd25f13c883f51",   ← SENT
  ...
}

Response: 200 OK
```

**GET Request (Retrieval):**
```bash
GET /rest-device/v2/device/?serial=UTD416000978&NetworkName=gch-control-only

Response:
{
  "_id": "689fcb2d03af0e3de789c742",
  "serial": "UTD416000978",
  "name": "UTD416000978",
  "NetworkName": "gch-control-only",
  "setupId": "",                            ← EMPTY!
  ...
}
```

### Impact

- **Device association succeeds** but **cannot be verified** via GET
- **List operations** cannot show which devices are associated with which setups
- **Verification tools** cannot confirm associations
- Documentation suggests `setupId` should be returned, but it's not

### Workaround

The SDK implements a workaround in `UpdateDevice()`:
- After PUT, if the response doesn't include `setupId`
- The SDK populates it from the request that was sent
- This provides immediate feedback but doesn't persist across sessions

```go
// After API update, if setupId is empty, populate from request
if response.Result.SetupID == "" && request.SetupID != "" {
    response.Result.SetupID = request.SetupID
}
```

### Questions for BrightSign

1. Is `setupId` intentionally excluded from GET responses?
2. Is there a query parameter or header to request it?
3. Is there a different endpoint that returns association information?
4. How do BrightSign players retrieve their `setupId` during boot if GET doesn't return it?

### Related Documentation

- `/docs/bdeploy-association.md` (line 84) states:
  `"B-Deploy returns device record with setupId"`
- API docs `/external/bs-api-docs-20250614/Cloud-APIs/B-Deploy_Provisioning-APIs/Version-2_0-B-Deploy-Endpoints/B-Deploy-Device-Endpoints-_v2_.md` (lines 42-49) do NOT list `setupId` as a returned field

### Status

**Investigating** - Waiting for clarification on whether this is:
- API limitation
- Documentation issue
- Bug
- Requires specific authentication or parameters
