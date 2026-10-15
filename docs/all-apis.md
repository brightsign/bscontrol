# BSN.cloud and B-Deploy API Endpoints - Comprehensive List

## Summary
- **BSN.cloud Main APIs (2022/06)**: 17 service categories, ~280 endpoints
- **B-Deploy Provisioning APIs**: 3 service categories (v2 and v3), 14 endpoints
- **Total**: ~294 endpoints
- **SDK Implementation Status**: 44 endpoints DONE (excluding removed content/presentation/upload)

**Legend:**
- `[DONE]` - API endpoint implemented in SDK with example CLI
- `[NOT-DONE]` - API endpoint not yet implemented

---

# BSN.cloud Main APIs (2022/06 version)

## Autoruns/Plugins
**Base URL:** `https://api.bsn.cloud/2022/06/REST/Autoruns/Plugins`

- `[NOT-DONE]` `GET /` - Returns a list of autorun plugins on a network
- `[NOT-DONE]` `POST /` - Create a new autorun plugin on a network
- `[NOT-DONE]` `DELETE /` - Removes autorun plugins matching the specified filter expression from a network
- `[NOT-DONE]` `GET /Count/` - Returns the number of autorun plugins matching the specified filter expression
- `[NOT-DONE]` `GET /{id:int}/` - Returns the specified autorun plugin on a network
- `[NOT-DONE]` `PUT /{id:int}/` - Update a specified autorun plugin on a network
- `[NOT-DONE]` `DELETE /{id:int}/` - Remove a specified autorun plugin from a network

## Device Subscriptions
**Base URL:** `https://api.bsn.cloud/2022/06/REST/Subscriptions`

- `[DONE]` `GET /` - Returns a list of device subscriptions (Example: `main-subscriptions-list`)
- `[DONE]` `GET /Count/` - Retrieves the number of subscription instances on the network (Example: `main-subscription-count`)
- `[DONE]` `GET /Operations/` - Returns operational permissions granted to roles (Example: `main-subscription-operations`)

## DeviceWebPages
**Base URL:** `https://api.bsn.cloud/2022/06/REST/DeviceWebPages`

- `[NOT-DONE]` `GET /` - Returns a list of device webpages
- `[NOT-DONE]` `DELETE /` - Removes device webpages, specified by a filter, from a network
- `[NOT-DONE]` `GET /Count/` - Returns the number of device webpages on a device
- `[NOT-DONE]` `GET /{id:int}/` - Returns the specified device webpage
- `[NOT-DONE]` `DELETE /{id:int}/` - Deletes the specified device webpage
- `[NOT-DONE]` `GET /{name}/` - Returns the specified device webpage
- `[NOT-DONE]` `DELETE /{name}/` - Deletes the specified device webpage
- `[NOT-DONE]` `GET /Operations/` - Returns operational permissions granted to roles
- `[NOT-DONE]` `GET /{id:int}/Permissions/` - Includes object permissions for a given device webpage instance
- `[NOT-DONE]` `POST /{id:int}/Permissions/` - Adds permissions for a device webpage
- `[NOT-DONE]` `DELETE /{id:int}/Permissions/` - Removes permissions from a device webpage
- `[NOT-DONE]` `GET /{name}/Permissions/` - Includes object permissions for a given device webpage instance
- `[NOT-DONE]` `POST /{name}/Permissions/` - Adds permissions for a device webpage
- `[NOT-DONE]` `DELETE /{name}/Permissions/` - Removes permissions from a device webpage

## Devices
**Base URL:** `https://api.bsn.cloud/2022/06/REST/Devices`

- `[DONE]` `GET /` - Retrieves a list of devices on the network (Example: `main--devices-list`)
- `[NOT-DONE]` `DELETE /` - Removes devices, specified by a filter, from a network
- `[NOT-DONE]` `GET /Regions/{*locationPath}/` - Returns a region containing all players in the specified path
- `[NOT-DONE]` `GET /Count/` - Retrieves the number of devices on the network
- `[DONE]` `GET /{id:int}/` - Return the information for a specified device (Example: `main-device-info`)
- `[DONE]` `PUT /{id:int}/` - Update the specified device (Example: `main-device-change-group`)
- `[NOT-DONE]` `PATCH /{id:int}/` - Replaces certain parameters on the specified device
- `[DONE]` `DELETE /{id:int}/` - Delete the specified device (Example: `main-device-delete`)
- `[DONE]` `GET /{serial}/` - Return the specified device information (Example: `main-device-info`)
- `[NOT-DONE]` `PUT /{serial}/` - Update a specified device
- `[NOT-DONE]` `PATCH /{serial}/` - Replaces certain parameters on the specified device
- `[DONE]` `DELETE /{serial}/` - Delete the specified device (Example: `main-device-delete`)
- `[NOT-DONE]` `GET /{deviceId:int}/Beacons/` - Return array of all beacons for the player
- `[NOT-DONE]` `GET /{serial}/Beacons/` - Return array of all beacons for the player
- `[NOT-DONE]` `GET /{deviceId:int}/Beacons/{name}/` - Returns a device beacon associated with specified device
- `[NOT-DONE]` `DELETE /{deviceId:int}/Beacons/{name}/` - Delete a specified device beacon
- `[NOT-DONE]` `GET /{serial}/Beacons/{name}/` - Returns a device beacon associated with specified device
- `[NOT-DONE]` `DELETE /{serial}/Beacons/{name}/` - Delete a specified device beacon
- `[NOT-DONE]` `POST /{id:int}/Beacons/` - Create a device beacon
- `[NOT-DONE]` `POST /{serial}/Beacons/` - Create a device beacon on a specified device
- `[DONE]` `GET /{id:int}/Errors/` - Returns a list of errors associated with a specified device (Example: `main-device-errors`)
- `[DONE]` `GET /{serial}/Errors/` - Returns a list of errors associated with a specified device (Example: `main-device-errors`)
- `[DONE]` `GET /{id:int}/Downloads/` - Returns the downloads associated with a specified device (Example: `main-device-downloads`)
- `[DONE]` `GET /{serial}/Downloads/` - Returns a list of downloads carried out by a device (Example: `main-device-downloads`)
- `[NOT-DONE]` `GET /{deviceid:int}/ScreenShots/` - Returns a list of screenshots uploaded by a specified device
- `[NOT-DONE]` `GET /{serial}/ScreenShots/` - Returns a list of screenshots uploaded by a specified device
- `[NOT-DONE]` `GET /ScreenShots/` - Returns a list of screenshots uploaded by a device
- `[NOT-DONE]` `GET /{id:int}/Tags/` - Returns tags on a device
- `[NOT-DONE]` `POST /{id:int}/Tags/` - Adds tags to a specified device
- `[NOT-DONE]` `DELETE /{id:int}/Tags/` - Remove tags from a specified device
- `[NOT-DONE]` `GET /{serial}/Tags/` - Returns tags on a device
- `[NOT-DONE]` `POST /{serial}/Tags/` - Adds one or more tags to a specified device
- `[NOT-DONE]` `DELETE /{serial}/Tags/` - Removes one or more tags from a specified device
- `[NOT-DONE]` `GET /Models/` - Returns the list of player models supported
- `[NOT-DONE]` `GET /Models/{model}/` - Returns the list of player models supported
- `[NOT-DONE]` `GET /Models/{model}/Connectors/` - Returns the list of connectors available on a device model
- `[NOT-DONE]` `GET /Models/{model}/Connectors/{connector}/` - Returns the list of connectors available
- `[NOT-DONE]` `GET /Models/{model}/Connectors/{connector}/VideoModes/` - Returns the video modes supported
- `[NOT-DONE]` `GET /Operations/` - Returns operational permissions granted to roles
- `[NOT-DONE]` `GET /{id:int}/Permissions/` - Returns permissions for the specified device
- `[NOT-DONE]` `POST /{id:int}/Permissions/` - Applies permissions to a specified device
- `[NOT-DONE]` `DELETE /{id:int}/Permissions/` - Removes custom permissions from a specified device
- `[NOT-DONE]` `GET /{serial}/Permissions/` - Returns permissions for the specified device
- `[NOT-DONE]` `POST /{serial}/Permissions/` - Applies permissions to a specified device
- `[NOT-DONE]` `DELETE /{serial}/Permissions/` - Removes custom permissions from a specified device
- `[NOT-DONE]` `GET /{serial}/Tokens/{token}/` - Validates an OAuth2 device access or refresh token
- `[NOT-DONE]` `DELETE /{serial}/Tokens/{token}/` - Revokes an OAuth2 device access or refresh token
- `[NOT-DONE]` `GET /{id:int}/Tokens/{token}/` - Validates an OAuth2 device access or refresh token
- `[NOT-DONE]` `DELETE /{id:int}/Tokens/{token}/` - Revokes an OAuth2 device access or refresh token
- `[NOT-DONE]` `GET /All/Notes/` - Retrieves a list of notes for all players on the network
- `[NOT-DONE]` `GET /{id:int}/Notes/` - Return the notes for a specified player
- `[NOT-DONE]` `PUT /{id:int}/Notes/` - Updates the notes for the specified player
- `[NOT-DONE]` `GET /{serial}/Notes/` - Return the notes for a specified player
- `[NOT-DONE]` `PUT /{serial}/Notes/` - Updates the notes for the specified player

## Feeds/Media
**Base URL:** `https://api.bsn.cloud/2022/06/REST/Feeds/Media`

- `[NOT-DONE]` `GET /` - Returns a list of live media feeds on a network
- `[NOT-DONE]` `POST /` - Creates a live media feed on a network
- `[NOT-DONE]` `DELETE /` - Removes live media feed instances, specified by a filter
- `[NOT-DONE]` `GET /Count/` - Returns the number of live media feeds on the network
- `[NOT-DONE]` `GET /{id:int}/` - Returns the specified live media feeds instance
- `[NOT-DONE]` `PUT /{id:int}/` - Modifies the specified live media feed instance
- `[NOT-DONE]` `DELETE /{id:int}/` - Removes the specified live media feed instance
- `[NOT-DONE]` `GET /{name}/` - Returns the specified live media feeds instance
- `[NOT-DONE]` `PUT /{name}/` - Modifies the specified live media feed instance
- `[NOT-DONE]` `DELETE /{name}/` - Removes the specified live media feed instance
- `[NOT-DONE]` `GET /Operations/` - Returns operational permissions granted to roles
- `[NOT-DONE]` `GET /{id:int}/Permissions/` - Includes object permissions for a given live media feed
- `[NOT-DONE]` `POST /{id:int}/Permissions/` - Adds permissions to live media feeds instance
- `[NOT-DONE]` `DELETE /{id:int}/Permissions/` - Removes permissions from live media feed instance
- `[NOT-DONE]` `GET /{name}/Permissions/` - Includes object permissions for a given live media feed
- `[NOT-DONE]` `POST /{name}/Permissions/` - Adds permissions to live media feeds instance
- `[NOT-DONE]` `DELETE /{name}/Permissions/` - Removes permissions from live media feed instance

## Feeds/Text
**Base URL:** `https://api.bsn.cloud/2022/06/REST/Feeds/Text`

- `[NOT-DONE]` `GET /` - Returns a list of live text feeds on a network
- `[NOT-DONE]` `POST /` - Creates a live text feed on a network
- `[NOT-DONE]` `DELETE /` - Removes live text feed instances, specified by a filter
- `[NOT-DONE]` `GET /Count/` - Returns the number of live text feeds on the network
- `[NOT-DONE]` `GET /{id:int}/` - Returns the specified live text feeds instance
- `[NOT-DONE]` `PUT /{id:int}/` - Modifies the specified live text feed instance
- `[NOT-DONE]` `DELETE /{id:int}/` - Removes the specified live text feed instance
- `[NOT-DONE]` `GET /{name}/` - Returns the specified live text feeds instance
- `[NOT-DONE]` `PUT /{name}/` - Modifies the specified live text feed instance
- `[NOT-DONE]` `DELETE /{name}/` - Removes the specified live text feed instance
- `[NOT-DONE]` `GET /Operations/` - Returns operational permissions granted to roles
- `[NOT-DONE]` `GET /{id:int}/Permissions/` - Includes object permissions for a given live text feed
- `[NOT-DONE]` `POST /{id:int}/Permissions/` - Adds permissions to live text feeds instance
- `[NOT-DONE]` `DELETE /{id:int}/Permissions/` - Removes permissions from live text feed instance
- `[NOT-DONE]` `GET /{name}/Permissions/` - Includes object permissions for a given live text feed
- `[NOT-DONE]` `POST /{name}/Permissions/` - Adds permissions to live text feeds instance
- `[NOT-DONE]` `DELETE /{name}/Permissions/` - Removes permissions from live text feed instance

## Groups/Regular
**Base URL:** `https://api.bsn.cloud/2022/06/REST/Groups/Regular`

- `[DONE]` `GET /` - Retrieves a list of groups on the network (Example: `main-device-change-group`)
- `[DONE]` `POST /` - Creates a group on the network (Example: `main-device-change-group`)
- `[NOT-DONE]` `DELETE /` - Removes groups, specified by a filter, from a network
- `[NOT-DONE]` `GET /Count/` - Returns the number of groups on the network
- `[DONE]` `GET /{id:int}/` - Returns a specified group (Example: `main-group-info`)
- `[DONE]` `PUT /{id:int}/` - Updates the specified group (Example: `main-group-update`)
- `[NOT-DONE]` `PATCH /{id:int}/` - Applies a sequence of changes to a specific group entity
- `[DONE]` `DELETE /{id:int}/` - Removes the specified group (Example: `main-group-delete`)
- `[NOT-DONE]` `GET /{name}/` - Returns a specified group
- `[NOT-DONE]` `PUT /{name}/` - Updates a specified group
- `[NOT-DONE]` `PATCH /{name}/` - Applies a sequence of changes to a specific group entity
- `[NOT-DONE]` `DELETE /{name}/` - Removes a specified group
- `[NOT-DONE]` `GET /{Id:int}/Schedule/` - Returns a list of scheduled presentations in the specified group
- `[NOT-DONE]` `POST /{Id:int}/Schedule/` - Adds a scheduled presentation to the specified group
- `[NOT-DONE]` `GET /{name}/Schedule/` - Returns a list of scheduled presentations in the specified group
- `[NOT-DONE]` `POST /{name}/Schedule/` - Adds a scheduled presentation to the specified group
- `[NOT-DONE]` `GET /{id:int}/Schedule/{scheduledPresentationId:int}/` - Returns the schedule of the specified presentation
- `[NOT-DONE]` `PUT /{id:int}/Schedule/{scheduledPresentationId:int}/` - Updates the specified scheduled presentation
- `[NOT-DONE]` `DELETE /{id:int}/Schedule/{scheduledPresentationId:int}/` - Removes a specified scheduled presentation
- `[NOT-DONE]` `GET /{name}/Schedule/{scheduledPresentationId:int}/` - Returns the schedule of the specified presentation
- `[NOT-DONE]` `PUT /{name}/Schedule/{scheduledPresentationId:int}/` - Updates the specified scheduled presentation
- `[NOT-DONE]` `DELETE /{name}/Schedule/{scheduledPresentationId:int}/` - Removes the specified scheduled presentation
- `[NOT-DONE]` `GET /Operations/` - Returns operational permissions granted to roles
- `[NOT-DONE]` `GET /{id:int}/Permissions/` - Includes object permissions for a given group instance
- `[NOT-DONE]` `POST /{id:int}/Permissions/` - Adds permissions to the specified group
- `[NOT-DONE]` `DELETE /{id:int}/Permissions/` - Deletes permissions from the specified group
- `[NOT-DONE]` `GET /{name}/Permissions/` - Includes object permissions for a given group instance
- `[NOT-DONE]` `POST /{name}/Permissions/` - Adds permissions to the specified group
- `[NOT-DONE]` `DELETE /{name}/Permissions/` - Removes permissions from the specified group

## Groups/Tagged
**Base URL:** `https://api.bsn.cloud/2022/06/REST/Groups/Tagged`

- `[NOT-DONE]` `GET /` - Retrieves a list of tagged groups on the network
- `[NOT-DONE]` `POST /` - Creates a tagged group on a network
- `[NOT-DONE]` `DELETE /` - Removes tagged groups, specified by a filter, from a network
- `[NOT-DONE]` `GET /Count/` - Returns the number of tagged groups on the network
- `[NOT-DONE]` `GET /{id:int}/` - Returns the specified tagged group
- `[NOT-DONE]` `PUT /{id:int}/` - Updates the specified tagged group
- `[NOT-DONE]` `DELETE /{id:int}/` - Deletes a specified tagged group
- `[NOT-DONE]` `GET /{name}/` - Returns the specified tagged group
- `[NOT-DONE]` `PUT /{name}/` - Updates the specified tagged group
- `[NOT-DONE]` `DELETE /{name}/` - Deletes a specified tagged group
- `[NOT-DONE]` `GET /Operations/` - Returns operational permissions granted to roles
- `[NOT-DONE]` `GET /{id:int}/Permissions/` - Includes object permissions for a given tagged group
- `[NOT-DONE]` `POST /{id:int}/Permissions/` - Adds permissions for a specified tagged group
- `[NOT-DONE]` `DELETE /{id:int}/Permissions/` - Removes permissions for a specified tagged group
- `[NOT-DONE]` `GET /{name}/Permissions/` - Includes object permissions for a given tagged group
- `[NOT-DONE]` `POST /{name}/Permissions/` - Adds permissions for a specified tagged group
- `[NOT-DONE]` `DELETE /{name}/Permissions/` - Removes permissions for a specified tagged group

## Playlists/Dynamic
**Base URL:** `https://api.bsn.cloud/2022/06/REST/Playlists/Dynamic`

- `[NOT-DONE]` `GET /` - Retrieves a list of dynamic playlists on the network
- `[NOT-DONE]` `POST /` - Create a new dynamic playlist on the network
- `[NOT-DONE]` `DELETE /` - Removes dynamic playlists, specified by a filter, from a network
- `[NOT-DONE]` `GET /Count/` - Returns the number of dynamic playlists on the network
- `[NOT-DONE]` `GET /{id:int}/` - Returns the specified dynamic playlist instance
- `[NOT-DONE]` `PUT /{id:int}/` - Modifies the specified dynamic playlist instance
- `[NOT-DONE]` `DELETE /{id:int}/` - Removes the specified dynamic playlist
- `[NOT-DONE]` `GET /{name}/` - Returns the specified dynamic playlist instance
- `[NOT-DONE]` `PUT /{name}/` - Modifies the specified dynamic playlist instance
- `[NOT-DONE]` `DELETE /{name}/` - Removes the specified dynamic playlist
- `[NOT-DONE]` `GET /Operations/` - Returns operational permissions granted to roles
- `[NOT-DONE]` `GET /{id:int}/Permissions/` - Includes object permissions for a given dynamic playlist
- `[NOT-DONE]` `POST /{id:int}/Permissions/` - Adds permissions to the specified dynamic playlist instance
- `[NOT-DONE]` `DELETE /{id:int}/Permissions/` - Removes permissions for the specified dynamic playlist
- `[NOT-DONE]` `GET /{name}/Permissions/` - Includes object permissions for a given dynamic playlist
- `[NOT-DONE]` `POST /{name}/Permissions/` - Adds permissions to the specified dynamic playlist instance
- `[NOT-DONE]` `DELETE /{name}/Permissions/` - Removes permissions for the specified dynamic playlist

## Playlists/Tagged
**Base URL:** `https://api.bsn.cloud/2022/06/REST/Playlists/Tagged`

- `[NOT-DONE]` `GET /` - Returns a list of tagged playlists on the network
- `[NOT-DONE]` `POST /` - Creates a tagged playlist on the network
- `[NOT-DONE]` `DELETE /` - Removes tagged playlists, specified by a filter
- `[NOT-DONE]` `GET /Count/` - Returns the number of tagged playlists on the network
- `[NOT-DONE]` `GET /{id:int}/` - Returns the specified tagged playlist instance
- `[NOT-DONE]` `PUT /{id:int}/` - Modifies the specified tagged playlist instance
- `[NOT-DONE]` `DELETE /{id:int}/` - Removes the specified tagged playlist
- `[NOT-DONE]` `GET /{name}/` - Returns the specified tagged playlist instance
- `[NOT-DONE]` `PUT /{name}/` - Modifies the specified tagged playlist instance
- `[NOT-DONE]` `DELETE /{name}/` - Removes the specified tagged playlist
- `[NOT-DONE]` `GET /Operations/` - Returns operational permissions granted to roles
- `[NOT-DONE]` `GET /{id:int}/Permissions/` - Includes object permissions for a given tagged playlist
- `[NOT-DONE]` `POST /{id:int}/Permissions/` - Retrieves permissions for the specified tagged playlist
- `[NOT-DONE]` `DELETE /{id:int}/Permissions/` - Removes permissions for the specified tagged playlist
- `[NOT-DONE]` `GET /{name}/Permissions/` - Includes object permissions for a given tagged playlist
- `[NOT-DONE]` `POST /{name}/Permissions/` - Retrieves permissions for the specified tagged playlist
- `[NOT-DONE]` `DELETE /{name}/Permissions/` - Removes permissions for the specified tagged playlist

## Provisioning
**Base URL:** `https://api.bsn.cloud/2022/06/REST/Provisioning`

- `[DONE]` `POST /Setups/Tokens/` - Issues a token for player registration in the current network (Example: `main-get-regtoken`)
- `[DONE]` `GET /Setups/Tokens/{token}/` - Validates a player setup token on the current network (Example: `main-get-regtoken`)
- `[DONE]` `DELETE /Setups/Tokens/{token}/` - Revokes a player setup token on the current network (Example: `main-revoke-regtoken`)

## Roles
**Base URL:** `https://api.bsn.cloud/2022/06/REST/Roles`

- `[NOT-DONE]` `GET /` - Returns a list of roles on a network
- `[NOT-DONE]` `POST /` - Creates a role on a network
- `[NOT-DONE]` `GET /{id:int}/` - Returns a specified role on a network
- `[NOT-DONE]` `PUT /{id:int}/` - Updates the specified role on a network
- `[NOT-DONE]` `DELETE /{id:int}/` - Removes the specified custom role on a network
- `[NOT-DONE]` `GET /{name}/` - Returns the specified role on a network
- `[NOT-DONE]` `PUT /{name}/` - Updates the specified role on a network
- `[NOT-DONE]` `DELETE /{name}/` - Removes the specified custom role on a network
- `[NOT-DONE]` `GET /Operations/` - Returns operational permissions granted to roles
- `[NOT-DONE]` `GET /{id:int}/Permissions/` - Includes object permissions granted to a given role
- `[NOT-DONE]` `POST /{id:int}/Permissions/` - Add permissions for specified roles on a network
- `[NOT-DONE]` `DELETE /{id:int}/Permissions/` - Removes permissions for specified roles on a network
- `[NOT-DONE]` `GET /{name}/Permissions/` - Includes object permissions granted to a given role
- `[NOT-DONE]` `POST /{name}/Permissions/` - Add permissions for specified roles on a network
- `[NOT-DONE]` `DELETE /{name}/Permissions/` - Removes permissions for specified role on a network

## Self (Personal/Session Management)
**Base URL:** `https://api.bsn.cloud/2022/06/REST`

- `[NOT-DONE]` `GET /Self/` - Returns your Person Entity information
- `[NOT-DONE]` `POST /Self/` - Registers the person and returns credentials
- `[NOT-DONE]` `GET /Self/Profile/` - Returns a complete person profile
- `[NOT-DONE]` `POST /Self/Profile/` - Creates a new profile property for a person
- `[NOT-DONE]` `GET /Self/Profile/{key}/` - Returns a profile property value for a person
- `[NOT-DONE]` `PUT /Self/Profile/{key}/` - Updates a profile property for a person
- `[NOT-DONE]` `DELETE /Self/Profile/{key}/` - Removes a profile property for a person
- `[NOT-DONE]` `GET /Self/Session/` - Retrieves complete set of attributes in the current session
- `[NOT-DONE]` `GET /Self/Session/Network/` - Retrieves network identifiers user is signed into
- `[NOT-DONE]` `GET /Self/Session/AuthorizationScope/` - Retrieves authorized action scope
- `[DONE]` `PUT /Self/Session/Network/` - Allows person to set or change network in current session (Used internally for network context)
- `[NOT-DONE]` `PUT /Self/Session/AuthorizationScope/` - Allows person to change authorized resources list
- `[NOT-DONE]` `GET /Self/Tokens/{token}/` - Gets status of specified OAuth2 person token
- `[NOT-DONE]` `DELETE /Self/Tokens/{token}/` - Revokes a person access or refresh token
- `[NOT-DONE]` `GET /Self/Networks/` - Returns networks associated with a person
- `[NOT-DONE]` `POST /Self/Networks/` - Creates a network for the person
- `[NOT-DONE]` `GET /Self/Networks/{id:int}/` - Get network associated with specified id
- `[NOT-DONE]` `PATCH /Self/Networks/{networkId:int}/` - Applies changes to network with specified id
- `[NOT-DONE]` `GET /Self/Networks/{name}/` - Returns network associated with specified name
- `[NOT-DONE]` `PATCH /Self/Networks/{networkName}/` - Applies changes to network with specified name
- `[NOT-DONE]` `GET /Self/Networks/{id:int}/Settings/` - Returns settings for specified network
- `[NOT-DONE]` `PUT /Self/Networks/{id:int}/Settings/` - Update settings for specified network
- `[NOT-DONE]` `GET /Self/Networks/{name}/Settings/` - Get settings for specified network
- `[NOT-DONE]` `PUT /Self/Networks/{name}/Settings/` - Update settings for specified network
- `[NOT-DONE]` `GET /Self/Networks/{id:int}/Subscription/` - Returns current subscription information
- `[NOT-DONE]` `PUT /Self/Networks/{id:int}/Subscription/` - Updates current subscription information
- `[NOT-DONE]` `GET /Self/Networks/{name}/Subscription/` - Returns current subscription information
- `[NOT-DONE]` `PUT /Self/Networks/{name}/Subscription/` - Updates current subscription information
- `[NOT-DONE]` `GET /Self/Networks/{id:int}/Subscriptions/` - Returns current and expired subscriptions
- `[NOT-DONE]` `GET /Self/Networks/{name}/Subscriptions/` - Returns current and expired subscriptions
- `[NOT-DONE]` `GET /Self/Users/` - Returns all user entities person is associated with
- `[NOT-DONE]` `GET /Self/Users/{id:int}/Role/` - Returns role information in a network
- `[NOT-DONE]` `GET /Self/Users/{id:int}/Profile/` - Returns user profile settings
- `[NOT-DONE]` `POST /Self/Users/{id:int}/Profile/` - Creates new user profile property
- `[NOT-DONE]` `GET /Self/Users/{id:int}/Profile/{key}/` - Returns value of user profile property
- `[NOT-DONE]` `PUT /Self/Users/{id:int}/Profile/{key}/` - Creates or updates user profile property
- `[NOT-DONE]` `DELETE /Self/Users/{id:int}/Profile/{key}/` - Removes property from user profile
- `[NOT-DONE]` `GET /Self/Users/{id:int}/Permissions/` - Returns permissions granted to user
- `[NOT-DONE]` `GET /Self/Users/{id:int}/Role/Permissions/` - Returns permissions granted to user
- `[NOT-DONE]` `GET /Self/Users/{id:int}/Notifications/` - Returns user notification settings
- `[NOT-DONE]` `PUT /Self/Users/{id:int}/Notifications/` - Updates user notification settings
- `[NOT-DONE]` `GET /Self/Applications/` - Returns pages of registered applications
- `[NOT-DONE]` `GET /Self/Applications/Scopes/` - Retrieves available scope tokens list
- `[NOT-DONE]` `POST /Self/Applications/` - Registers new application for OAuth2 credentials
- `[NOT-DONE]` `DELETE /Self/Applications/` - Unregisters unused Application instances
- `[NOT-DONE]` `GET /Self/Applications/Count/` - Returns number of registered application instances
- `[NOT-DONE]` `GET /Self/Applications/{id}/` - Returns information about registered Application
- `[NOT-DONE]` `PUT /Self/Applications/{id}/` - Edits information about application
- `[NOT-DONE]` `DELETE /Self/Applications/{id}/` - Unregisters unused Application instance
- `[NOT-DONE]` `POST /Self/Applications/{id}/Secret/` - Rotates OAuth2 Client Secret

## Tags
**Base URL:** `https://api.bsn.cloud/2022/06/REST/Tags`

- `[NOT-DONE]` `GET /Keys/` - Returns all tag names defined on network that match pattern
- `[NOT-DONE]` `GET /Values/` - Returns all tag values defined on network that match pattern

## Users
**Base URL:** `https://api.bsn.cloud/2022/06/REST/Users`

- `[NOT-DONE]` `GET /` - Returns a list of user instances on a network
- `[NOT-DONE]` `POST /` - Creates a user instance on a network
- `[NOT-DONE]` `GET /{login}/` - Returns information for specified user on network
- `[NOT-DONE]` `PUT /{login}/` - Updates information for specified user on network
- `[NOT-DONE]` `DELETE /{login}/` - Deletes specified user on network
- `[NOT-DONE]` `GET /{id:int}/` - Returns information for specified user on network
- `[NOT-DONE]` `PUT /{id:int}/` - Update given user instance
- `[NOT-DONE]` `DELETE /{id:int}/` - Deletes specified user on network
- `[NOT-DONE]` `GET /Operations/` - Returns operational permissions granted to roles
- `[NOT-DONE]` `GET /{id:int}/Permissions/` - Includes object permissions for given user
- `[NOT-DONE]` `POST /{id:int}/Permissions/` - Adds permissions for specified user on network
- `[NOT-DONE]` `DELETE /{id:int}/Permissions/` - Removes permissions for specified user on network
- `[NOT-DONE]` `GET /{login}/Permissions/` - Includes object permissions for given user
- `[NOT-DONE]` `POST /{login}/Permissions/` - Adds permissions for specified user on network
- `[NOT-DONE]` `DELETE /{login}/Permissions/` - Removes permissions for specified user on network
- `[NOT-DONE]` `GET /{id:int}/Tokens/{token}/` - Validates user access or refresh token
- `[NOT-DONE]` `DELETE /{id:int}/Tokens/{token}/` - Revokes user access or refresh tokens
- `[NOT-DONE]` `GET /{login}/Tokens/{token}/` - Validates user access or refresh token
- `[NOT-DONE]` `DELETE /{login}/Tokens/{token}/` - Revokes user access or refresh token

## Web Application
**Base URL:** `https://api.bsn.cloud/2022/06/REST/WebApplications`

- `[NOT-DONE]` `GET /` - Returns list of web applications on network
- `[NOT-DONE]` `DELETE /` - Deletes web applications specified by filter
- `[NOT-DONE]` `GET /Count/` - Returns number of web applications on network
- `[NOT-DONE]` `GET /{applicationId}/` - Returns specified web application
- `[NOT-DONE]` `DELETE /{applicationId}/` - Deletes specified web application
- `[NOT-DONE]` `GET /{applicationName}/` - Returns specified web application
- `[NOT-DONE]` `DELETE /{applicationName}/` - Deletes specified web application
- `[NOT-DONE]` `GET /Operations/` - Returns operational permissions granted to roles

## WebPages
**Base URL:** `https://api.bsn.cloud/2022/06/REST/WebPages`

- `[NOT-DONE]` `GET /` - Returns list of webpages on network
- `[NOT-DONE]` `DELETE /` - Deletes webpages specified by filter
- `[NOT-DONE]` `GET /Count/` - Returns number of webpages on network
- `[NOT-DONE]` `GET /{id:int}/` - Returns specified webpage
- `[NOT-DONE]` `DELETE /{id:int}/` - Deletes specified webpage
- `[NOT-DONE]` `GET /{name}/` - Returns specified webpage
- `[NOT-DONE]` `DELETE /{name}/` - Deletes specified webpage
- `[NOT-DONE]` `GET /Operations/` - Returns operational permissions granted to roles
- `[NOT-DONE]` `GET /{id:int}/Permissions/` - Includes object permissions for given webpage
- `[NOT-DONE]` `POST /{id:int}/Permissions/` - Adds permissions for webpage on network
- `[NOT-DONE]` `DELETE /{id:int}/Permissions/` - Removes permissions from webpage
- `[NOT-DONE]` `GET /{name}/Permissions/` - Includes object permissions for given webpage
- `[NOT-DONE]` `POST /{name}/Permissions/` - Adds permissions for webpage on network
- `[NOT-DONE]` `DELETE /{name}/Permissions/` - Removes permissions from webpage

---

# Remote DWS (Diagnostic Web Server) APIs

**Base URL:** `https://ws.bsn.cloud/rest/v1/`

## Information Endpoints

- `[DONE]` `GET /info/` - Retrieves general information about the player (Example: `rdws-info`)
- `[DONE]` `GET /time/` - Retrieves the date and time configured on the player (Example: `rdws-time`)
- `[DONE]` `PUT /time/` - Sets the date/time on the player (Example: `rdws-time-set`)
- `[DONE]` `GET /health/` - Retrieves the current status of the player (Example: `rdws-health`)

## Files Endpoints

- `[DONE]` `GET /files/{:path}/` - Lists directories and/or files in a path (Example: `rdws-files-list`)
- `[DONE]` `PUT /files/{:path}` - Uploads a new file or folder to player storage (Examples: `rdws-files-upload`, `rdws-files-create-folder`)
- `[DONE]` `POST /files/{:path}/` - Renames a file in the specified path (Example: `rdws-files-rename`)
- `[DONE]` `DELETE /files/{:path}/` - Deletes a file from player storage (Example: `rdws-files-delete`)

## Control Endpoints

- `[DONE]` `PUT /control/reboot/` - Reboots the player (with optional crash report, factory reset, or disable autorun) (Example: `rdws-reboot`)
- `[DONE]` `GET /control/dws-password/` - Retrieves information about current local DWS password (Example: `rdws-dws-password`)
- `[DONE]` `PUT /control/dws-password/` - Sets a new password for local DWS (Example: `rdws-dws-password`)
- `[DONE]` `GET /control/local-dws/` - Retrieves current state of local DWS (Example: `rdws-local-dws`)
- `[DONE]` `PUT /control/local-dws/` - Enables or disables local DWS (Example: `rdws-local-dws`)

## Diagnostics Endpoints

- `[DONE]` `GET /diagnostics/` - Runs network diagnostics on the player (Example: `rdws-diagnostics`)
- `[DONE]` `GET /diagnostics/dns-lookup/{:domain_name}/` - Tests name resolution on specified DNS address (Example: `rdws-dns-lookup`)
- `[DONE]` `GET /diagnostics/ping/{:domain_name}/` - Pings specified IP or DNS address on local network (Example: `rdws-ping`)
- `[DONE]` `GET /diagnostics/trace-route/{:domain_name}/` - Performs trace-route diagnostic on specified IP or DNS address (Example: `rdws-traceroute`)
- `[DONE]` `GET /diagnostics/network-configuration/{:interface}/` - Retrieves network-interface settings (Example: `rdws-network-config`)
- `[DONE]` `PUT /diagnostics/network-configuration/{:interface}/` - Applies test network configuration (Example: `rdws-network-config`)
- `[DONE]` `GET /diagnostics/network-neighborhood/` - Retrieves information about player's network neighborhood (Example: `rdws-network-neighborhood`)
- `[DONE]` `GET /diagnostics/packet-capture/` - Gets current status of packet capture operation (Example: `rdws-packet-capture`)
- `[DONE]` `POST /diagnostics/packet-capture/` - Starts a packet capture operation (Example: `rdws-packet-capture`)
- `[DONE]` `DELETE /diagnostics/packet-capture/` - Stops a packet capture operation (Example: `rdws-packet-capture`)
- `[DONE]` `GET /diagnostics/telnet/` - Get telnet information (enabled status and port number) (Example: `rdws-telnet`)
- `[DONE]` `PUT /diagnostics/telnet/` - Enable/disable telnet on the player (Example: `rdws-telnet`)
- `[DONE]` `GET /diagnostics/ssh/` - Returns SSH information (enabled status and port number) (Example: `rdws-ssh`)
- `[DONE]` `PUT /diagnostics/ssh/` - Enable/disable SSH on the player (Example: `rdws-ssh`)

## Display Control Endpoints
*For Moka displays with built-in BrightSign players, BOS 9.0.189+*

- `[NOT-DONE]` `GET /v1/display-control/` - Returns all control settings for connected display
- `[NOT-DONE]` `GET /v1/display-control/brightness/` - Returns brightness settings
- `[NOT-DONE]` `PUT /v1/display-control/brightness/` - Changes brightness setting
- `[NOT-DONE]` `GET /v1/display-control/contrast/` - Returns contrast settings
- `[NOT-DONE]` `PUT /v1/display-control/contrast/` - Changes contrast setting
- `[NOT-DONE]` `GET /v1/display-control/always-connected/` - Returns connection settings
- `[NOT-DONE]` `PUT /v1/display-control/always-connected/` - Changes connection setting
- `[NOT-DONE]` `PUT /v1/display-control/firmware/` - Changes firmware setting
- `[NOT-DONE]` `GET /v1/display-control/info/` - Returns BrightSign player information
- `[NOT-DONE]` `GET /v1/display-control/power-settings/` - Returns power settings
- `[NOT-DONE]` `PUT /v1/display-control/power-settings/` - Changes power setting
- `[NOT-DONE]` `GET /v1/display-control/standby-timeout/` - Returns standby/timeout settings
- `[NOT-DONE]` `PUT /v1/display-control/standby-timeout/` - Changes standby/timeout setting
- `[NOT-DONE]` `GET /v1/display-control/sd-connection/` - Returns SD connection settings
- `[NOT-DONE]` `PUT /v1/display-control/sd-connection/` - Changes SD connection setting
- `[NOT-DONE]` `GET /v1/display-control/video-output/` - Returns video output settings
- `[NOT-DONE]` `PUT /v1/display-control/video-output/` - Changes video output setting
- `[NOT-DONE]` `GET /v1/display-control/volume/` - Returns volume settings
- `[NOT-DONE]` `PUT /v1/display-control/volume/` - Changes volume setting
- `[NOT-DONE]` `GET /v1/display-control/white-balance/` - Returns white balance settings
- `[NOT-DONE]` `PUT /v1/display-control/white-balance/` - Changes white balance setting

## Logs Endpoints

- `[DONE]` `GET /logs/` - Retrieves log files from the player (example: [rdws-logs-get](../examples/bsn-rdws-logs-get))
- `[DONE]` `GET /crash-dump/` - Retrieves crash dump files (example: [rdws-crashdump-get](../examples/bsn-rdws-crashdump-get))

## Storage & Provisioning Endpoints

- `[DONE]` `DELETE /storage/{:device_name}/` - Reformats specified storage device (Example: `rdws-reformat-storage`)
- `[DONE]` `GET /re-provision/` - Re-provisions the player with B-Deploy/Provisioning setup (Example: `rdws-reprovision`)
- `[DONE]` `POST /snapshot/` - Takes a screenshot and saves to storage (Example: `rdws-snapshot`)
- `[DONE]` `PUT /custom/` - Sends custom data to player via UDP port 5000 (Example: `rdws-custom-data`)
- `[DONE]` `GET /download-firmware/` - Downloads and applies firmware update (Example: `rdws-firmware-download`)

## Registry Endpoints

- `[DONE]` `GET /registry/` - Returns player registry dump (Example: `rdws-registry-get`)
- `[DONE]` `GET /registry/:section/:key/` - Returns particular registry section or key value (Example: `rdws-registry-get`)
- `[DONE]` `PUT /registry/:section/:key/` - Sets value in specified registry section (Example: `rdws-registry-set`)
- `[DONE]` `DELETE /registry/:section/:key/` - Deletes key-value pair from registry section (Example: `rdws-registry-set`)
- `[DONE]` `PUT /registry/flush/` - Flushes registry immediately to disk (Example: `rdws-registry-set`)
- `[DONE]` `GET /registry/recovery_url/` - Retrieves recovery URL from player registry (Example: `rdws-registry-get`)
- `[DONE]` `PUT /registry/recovery_url/` - Writes new recovery URL to player registry (Example: `rdws-registry-set`)

## Remoteview Endpoints

- `[NOT-DONE]` `GET /remoteview/config/` - Checks remoteview configuration
- `[NOT-DONE]` `PUT /remoteview/config/` - Configures player with access URL
- `[NOT-DONE]` `GET /remoteview/{:source}/view/` - Returns information about active remote view sessions
- `[NOT-DONE]` `GET /remoteview/{:source}/view/:id/` - Returns information about specified session
- `[NOT-DONE]` `POST /remoteview/{:source}/view/` - Starts a new remoteview session
- `[NOT-DONE]` `DELETE /remoteview/{:source}/view/:id/` - Stops specified remote view session

## Video Endpoints

- `[NOT-DONE]` `GET /video-mode/` - Retrieves currently active video mode
- `[NOT-DONE]` `GET /video/{:connector}/output/{:device}/` - Retrieves information about specified video output
- `[NOT-DONE]` `GET /video/{:connector}/output/{:device}/edid/` - Retrieves EDID information
- `[NOT-DONE]` `GET /video/{:connector}/output/{:device}/power-save/` - Returns power save status
- `[NOT-DONE]` `PUT /video/{:connector}/output/{:device}/power-save/` - Sets power save mode
- `[NOT-DONE]` `GET /video/{:connector}/output/{:device}/modes/` - Returns available video modes
- `[NOT-DONE]` `GET /video/{:connector}/output/{:device}/mode/` - Returns current video mode
- `[NOT-DONE]` `PUT /video/{:connector}/output/{:device}/mode/` - Sets video mode

---

# B-Deploy Provisioning APIs

## B-Deploy Device Endpoints (v2)
**Base URL:** `https://provision.bsn.cloud/rest-device/v2/device/`

- `[DONE]` `GET /` - Retrieves device object associated with B-Deploy account (Example: `bdeploy-list-devices`)
- `[DONE]` `GET /?_id={id}` - Retrieves specific device object by ID (Example: `bdeploy-get-device`)
- `[NOT-DONE]` `POST /` - Adds device object to B-Deploy account
- `[DONE]` `PUT /?_id={id}` - Modifies device object associated with B-Deploy account (Example: `bdeploy-associate`)
- `[DONE]` `DELETE /?_id={id}` - Removes device object from B-Deploy account (Example: `bdeploy-delete-device`)
- `[NOT-DONE]` `DELETE /?serial={serial}` - Removes device object by serial number

## B-Deploy Setup Endpoints (v2)
**Base URL:** `https://provision.bsn.cloud/rest-setup/v2/setup/`

- `[NOT-DONE]` `GET /` - Retrieves player setups for network and username
- `[NOT-DONE]` `GET /?_id={id}` - Retrieves specific player setup by ID
- `[NOT-DONE]` `POST /` - Adds new player setup to B-Deploy server
- `[NOT-DONE]` `PUT /?_id={id}` - Updates existing player setup on B-Deploy server
- `[NOT-DONE]` `DELETE /?_id={id}` - Deletes player setup from B-Deploy server

**Note**: SDK uses v3 endpoints instead of v2

## B-Deploy Setup Endpoints (v3)
**Base URL:** `https://provision.bsn.cloud/rest-setup/v3/setup/`

- `[DONE]` `GET /` - Returns list of setup records from B-Deploy database (Example: `bdeploy-list-setups`)
- `[DONE]` `POST /` - Adds device setups to B-Deploy server (Example: `bdeploy-add-setup`)
- `[DONE]` `PUT /` - Updates existing device setups on B-Deploy server (Example: `bdeploy-update-setup`)
- `[DONE]` `DELETE /?_id={id}` - Deletes device setup from B-Deploy server (Example: `bdeploy-delete-setup`)
- `[DONE]` `GET /{_id}/` - Retrieves device setup for specified setup id (Example: `bdeploy-get-setup`)

---

## Implementation Statistics

### BSN.cloud Main APIs (2022/06)
- **Implemented**: 17 endpoints
- **Not Implemented**: ~247 endpoints

**Breakdown by Category:**
- Autoruns/Plugins: 0/7 (0%)
- **Device Subscriptions: 3/3 (100%)** ✓
- DeviceWebPages: 0/14 (0%)
- **Devices: 9/54 (17%)** ✓
- Feeds/Media: 0/17 (0%)
- Feeds/Text: 0/17 (0%)
- **Groups/Regular: 3/27 (11%)** ✓
- Groups/Tagged: 0/17 (0%)
- Playlists/Dynamic: 0/17 (0%)
- Playlists/Tagged: 0/17 (0%)
- **Provisioning: 3/3 (100%)** ✓
- Roles: 0/16 (0%)
- **Self: 1/48 (2%)** ✓
- Tags: 0/2 (0%)
- Users: 0/20 (0%)
- Web Application: 0/8 (0%)
- WebPages: 0/14 (0%)

### B-Deploy Provisioning APIs
- **Implemented**: 10/14 endpoints (71%)
- **Not Implemented**: 4/14 endpoints (29%)

**Breakdown by Version:**
- Device (v2): 4/6 (67%)
- Setup (v2): 0/5 (0%) - SDK uses v3
- **Setup (v3): 5/5 (100%)** ✓

### Overall Summary
- **Total Endpoints**: ~294
- **Implemented with Examples**: 44
- **Not Implemented**: ~250

### Example Programs Available
Working CLI examples covering:
- **Main API** - Device management (list, info, status, errors, downloads, delete, update, change group)
- **Main API** - Group management (list, create, info, update, delete)
- **Main API** - Subscription management (list, count, operations)
- **Main API** - Token generation and validation
- **RDWS** - Control operations (reboot, snapshot, reprovision, DWS password, local DWS)
- **RDWS** - Remote diagnostics (info, time, health, file management)
- **RDWS** - Network diagnostics (ping, traceroute, DNS lookup, network config, neighborhood scan)
- **RDWS** - Packet capture and remote access (telnet, SSH)
- **RDWS** - Storage management (reformat storage devices)
- **RDWS** - Custom commands (send custom data via UDP port 5000)
- **RDWS** - Firmware management (download and apply firmware updates)
- **RDWS** - Registry management (get/set registry values, flush, recovery URL)
- **RDWS** - Logs and diagnostics (retrieve log files and crash dumps)
- **B-Deploy** - Provisioning (setup and device management)

---

## Notes on Path Parameters

- `{id:int}` - Integer identifier for the resource
- `{name}` - String name of the resource
- `{serial}` - Device serial number
- `{*virtualPath}` - Variable-length virtual path
- `{token}` - OAuth2 token string
- `{key}` - Profile key string
- `{model}` - Device model identifier
- `{connector}` - Device connector identifier

## Common Query Parameters

- `filter` - Expression for filtering search results
- `sort` - Expression for sorting results
- `marker` - Pagination marker for retrieving next page
- `pageSize` - Maximum number of items per page (default/max: 100)
- `NetworkName` - Network identifier
- `username` - User identifier
