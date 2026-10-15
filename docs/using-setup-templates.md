# Using Setup Templates

Setup templates are JSON files with Go `text/template` placeholders that get rendered into concrete device setup configurations. The template file is located at `setups/setup-template.json`.

## Template Placeholders

The following placeholders must be populated when rendering the template:

| Field Path | Placeholder | Description |
|---|---|---|
| `bDeploy.username` | `{{.Username}}` | BSN cloud username for the deployment |
| `bDeploy.networkName` | `{{.NetworkName}}` | BSN network name to associate the device with |
| `bDeploy.packageName` | `{{.PackageName}}` | Name of the deployment package |
| `bsnDeviceRegistrationTokenEntity.token` | `{{.RegistrationToken}}` | Device registration token value |
| `bsnDeviceRegistrationTokenEntity.validFrom` | `{{.TokenValidFrom}}` | Registration token validity start timestamp |
| `bsnDeviceRegistrationTokenEntity.validTo` | `{{.TokenValidTo}}` | Registration token validity end timestamp |
| `deviceName` | `{{.DeviceName}}` | Human-readable name assigned to the device |
| `deviceDescription` | `{{.DeviceDescription}}` | Description text for the device |
| `bsnGroupName` | `{{.GroupName}}` | BSN group the device belongs to |

## Template Structure

The template is a version 3.0.0 setup configuration with `setupType: "lfn"`. It contains:

- **bDeploy** -- Deployment metadata (username, network, client, package name). The `client` field is hardcoded to `"bacon"`.
- **firmwareUpdatesByFamily** -- Firmware definitions for 10 device families (Impala, Pantera, Tiger, Pagani, Malibu, Sebring, Raptor, Cobra, Camaro, Thor). Each family includes production, beta, and compatible release URLs, versions, version numbers, SHA1 hashes, and file lengths.
- **bsnDeviceRegistrationTokenEntity** -- Registration token with scope `"cert"` and templated validity window.
- **network** -- Structured network configuration with a single wired Ethernet interface using DHCPv4, WPA enterprise settings (disabled by default), and a time server.
- **logging** -- Playback, event, diagnostic, state, and variable logging all enabled. Log upload at boot and scheduled upload are disabled.
- **diagnostics** -- Serial and system log debugging enabled. Remote DWS enabled, local DWS and LWS disabled.
- **display** -- Idle screen color set to black. Remote snapshots disabled. No custom splash screen.

## Environment Variables

Three environment variables control template placeholder values. They follow the existing `BS_` naming convention:

| Environment Variable | Flag | Template Placeholder | Description |
|---|---|---|---|
| `BS_PACKAGE_NAME` | `--package-name` | `{{.PackageName}}` | Setup package identifier |
| `BS_DEVICE_NAME` | `--device-name` | `{{.DeviceName}}` | Human-readable device name |
| `BS_DEVICE_DESCRIPTION` | `--device-description` | `{{.DeviceDescription}}` | Device description text |

These work alongside the existing environment variables:

| Environment Variable | Description |
|---|---|
| `BS_CLIENT_ID` | BSN.cloud API client ID (also used as username) |
| `BS_SECRET` | BSN.cloud API client secret |
| `BS_NETWORK` | BSN.cloud network name |

**Resolution order:** Command-line flag > environment variable > empty string (or error if required).

## Usage by Example Programs

Each `bdeploy-*` example supports these variables as follows:

| Example | `--package-name` / `BS_PACKAGE_NAME` | `--device-name` / `BS_DEVICE_NAME` | `--device-description` / `BS_DEVICE_DESCRIPTION` |
|---|---|---|---|
| `bdeploy-add-setup` | required (template mode) | optional | optional |
| `bdeploy-list-setups` | optional (filter) | -- | -- |
| `bdeploy-get-setup` | optional (lookup) | -- | -- |
| `bdeploy-update-setup` | optional (update field) | -- | -- |
| `bdeploy-delete-setup` | optional (lookup) | -- | -- |
| `bdeploy-associate` | optional (lookup) | optional | -- |

### Template Mode (`bdeploy-add-setup`)

The `bdeploy-add-setup` example supports two modes:

**Template mode** (new) -- renders `setups/setup-template.json` with values from flags/env:

```bash
export BS_PACKAGE_NAME="retail-display-v1"
export BS_NETWORK="Production"
bdeploy-add-setup
```

Or with explicit flags:

```bash
bdeploy-add-setup --package-name "retail-v1" --network "Production" --group "Lobby"
```

With device info:

```bash
bdeploy-add-setup \
  --package-name "retail-v1" \
  --network "Production" \
  --device-name "Lobby Kiosk" \
  --device-description "Main entrance display"
```

Use `--template` to specify an alternative template file (default: `setups/setup-template.json`).

**Config file mode** (backward compatible) -- pass a JSON config file as a positional argument:

```bash
bdeploy-add-setup config.json
```

### Filter/Lookup Examples

The other examples use `--package-name` / `BS_PACKAGE_NAME` for filtering or lookup:

```bash
export BS_PACKAGE_NAME="retail-display-v1"

# List setups filtered by package name
bdeploy-list-setups --network "Production"

# Get a specific setup by package name
bdeploy-get-setup --package-name "retail-display-v1" --network "Production"

# Delete a setup by package name
bdeploy-delete-setup --package-name "retail-display-v1" --network "Production"

# Associate a device with a setup by package name
bdeploy-associate --serial ABC123 --package-name "retail-display-v1" --device-name "Lobby"
```

## Full Template Reference

Below is the complete `setups/setup-template.json` file:

```json
{
  "uiDeviceSetupErrors": [],
  "version": "3.0.0",
  "bDeploy": {
    "username": "{{.Username}}",
    "networkName": "{{.NetworkName}}",
    "client": "bacon",
    "packageName": "{{.PackageName}}"
  },
  "firmwareUpdatesByFamily": {
    "Impala": {
      "firmwareUpdateSource": null,
      "firmwareUpdateSourceFilePath": "",
      "firmwareUpdateSourceUrl": "",
      "firmwareUpdateStandardTargetFileName": "impala-update.bsfw",
      "firmwareUpdateDifferentTargetFileName": "impala-update_different.bsfw",
      "firmwareUpdateNewerTargetFileName": "impala-update_newer.bsfw",
      "firmwareUpdateSaveTargetFileName": "impala-update_save.bsfw",
      "firmwareUpdateVersion": "",
      "productionReleaseURL": "https://firmware.bsn.cloud/impala-8.5.64-update.bsfw",
      "betaReleaseURL": "https://firmware.bsn.cloud/impala-9.1.96-update.bsfw",
      "compatibleReleaseURL": "https://firmware.bsn.cloud/impala-8.0.146-update.bsfw",
      "productionVersion": "8.5.64",
      "betaVersion": "9.1.96",
      "compatibleVersion": "8.0.146",
      "productionVersionNumber": 525632,
      "betaVersionNumber": 590176,
      "compatibleVersionNumber": 524434,
      "productionReleaseSHA1": "cd245cf6b699ea943d7dd0a65359729c0046425b",
      "betaReleaseSHA1": "5f021b7e7bc7f153e5a3bfba2705de5c7a95743f",
      "compatibleReleaseSHA1": "789cf557fbb0a5f70ecc103e5d071186b57eb38f",
      "productionReleaseFileLength": 144974118,
      "betaReleaseFileLength": 175558114,
      "compatibleReleaseFileLength": 137811964,
      "existingFWContentID": ""
    },
    "Pantera": {
      "firmwareUpdateSource": null,
      "firmwareUpdateSourceFilePath": "",
      "firmwareUpdateSourceUrl": "",
      "firmwareUpdateStandardTargetFileName": "pantera-update.bsfw",
      "firmwareUpdateDifferentTargetFileName": "pantera-update_different.bsfw",
      "firmwareUpdateNewerTargetFileName": "pantera-update_newer.bsfw",
      "firmwareUpdateSaveTargetFileName": "pantera-update_save.bsfw",
      "firmwareUpdateVersion": "",
      "productionReleaseURL": "https://firmware.bsn.cloud/pantera-8.5.64-update.bsfw",
      "betaReleaseURL": "https://firmware.bsn.cloud/pantera-9.1.96-update.bsfw",
      "compatibleReleaseURL": "https://firmware.bsn.cloud/pantera-8.0.146-update.bsfw",
      "productionVersion": "8.5.64",
      "betaVersion": "9.1.96",
      "compatibleVersion": "8.0.146",
      "productionVersionNumber": 525632,
      "betaVersionNumber": 590176,
      "compatibleVersionNumber": 524434,
      "productionReleaseSHA1": "d60406cb2017b857878ef336bc237cfe32f89ed3",
      "betaReleaseSHA1": "ea9f44280a1d89155f8520e36b03c95b1730bc63",
      "compatibleReleaseSHA1": "58196ad8707b6e62185f1aba6d124ed1adfa7f67",
      "productionReleaseFileLength": 144431180,
      "betaReleaseFileLength": 174973584,
      "compatibleReleaseFileLength": 137509058,
      "existingFWContentID": ""
    },
    "Tiger": {
      "firmwareUpdateSource": null,
      "firmwareUpdateSourceFilePath": "",
      "firmwareUpdateSourceUrl": "",
      "firmwareUpdateStandardTargetFileName": "tiger-update.bsfw",
      "firmwareUpdateDifferentTargetFileName": "tiger-update_different.bsfw",
      "firmwareUpdateNewerTargetFileName": "tiger-update_newer.bsfw",
      "firmwareUpdateSaveTargetFileName": "tiger-update_save.bsfw",
      "firmwareUpdateVersion": "",
      "productionReleaseURL": "https://firmware.bsn.cloud/tiger-8.5.64-update.bsfw",
      "betaReleaseURL": "https://firmware.bsn.cloud/tiger-8.5.64-update.bsfw",
      "compatibleReleaseURL": "https://firmware.bsn.cloud/tiger-8.0.146-update.bsfw",
      "productionVersion": "8.5.64",
      "betaVersion": "8.5.64",
      "compatibleVersion": "8.0.146",
      "productionVersionNumber": 525632,
      "betaVersionNumber": 525632,
      "compatibleVersionNumber": 524434,
      "productionReleaseSHA1": "c18e4582b44dd610e93c0b23b1b65cbc8ee76bb8",
      "betaReleaseSHA1": "c18e4582b44dd610e93c0b23b1b65cbc8ee76bb8",
      "compatibleReleaseSHA1": "f02173d13f800d8d264d761226f783baa27b44b2",
      "productionReleaseFileLength": 156538762,
      "betaReleaseFileLength": 156538762,
      "compatibleReleaseFileLength": 147734484,
      "existingFWContentID": ""
    },
    "Pagani": {
      "firmwareUpdateSource": null,
      "firmwareUpdateSourceFilePath": "",
      "firmwareUpdateSourceUrl": "",
      "firmwareUpdateStandardTargetFileName": "pagani-update.bsfw",
      "firmwareUpdateDifferentTargetFileName": "pagani-update_different.bsfw",
      "firmwareUpdateNewerTargetFileName": "pagani-update_newer.bsfw",
      "firmwareUpdateSaveTargetFileName": "pagani-update_save.bsfw",
      "firmwareUpdateVersion": "",
      "productionReleaseURL": "https://firmware.bsn.cloud/pagani-8.5.64-update.bsfw",
      "betaReleaseURL": "https://firmware.bsn.cloud/pagani-9.1.96-update.bsfw",
      "compatibleReleaseURL": "https://firmware.bsn.cloud/pagani-8.0.146-update.bsfw",
      "productionVersion": "8.5.64",
      "betaVersion": "9.1.96",
      "compatibleVersion": "8.0.146",
      "productionVersionNumber": 525632,
      "betaVersionNumber": 590176,
      "compatibleVersionNumber": 524434,
      "productionReleaseSHA1": "8f22d047ff43a873264f0a63f523a4518d34999a",
      "betaReleaseSHA1": "36f274caf938a1fcfd282106913d216a3060dd85",
      "compatibleReleaseSHA1": "9af7416c055f9fe45d3b492ad00a7cb8b0e5f7ff",
      "productionReleaseFileLength": 200687450,
      "betaReleaseFileLength": 204679900,
      "compatibleReleaseFileLength": 167649562,
      "existingFWContentID": ""
    },
    "Malibu": {
      "firmwareUpdateSource": null,
      "firmwareUpdateSourceFilePath": "",
      "firmwareUpdateSourceUrl": "",
      "firmwareUpdateStandardTargetFileName": "malibu-update.bsfw",
      "firmwareUpdateDifferentTargetFileName": "malibu-update_different.bsfw",
      "firmwareUpdateNewerTargetFileName": "malibu-update_newer.bsfw",
      "firmwareUpdateSaveTargetFileName": "malibu-update_save.bsfw",
      "firmwareUpdateVersion": "",
      "productionReleaseURL": "https://firmware.bsn.cloud/malibu-8.5.64-update.bsfw",
      "betaReleaseURL": "https://firmware.bsn.cloud/malibu-9.1.96-update.bsfw",
      "compatibleReleaseURL": "https://firmware.bsn.cloud/malibu-8.0.146-update.bsfw",
      "productionVersion": "8.5.64",
      "betaVersion": "9.1.96",
      "compatibleVersion": "8.0.146",
      "productionVersionNumber": 525632,
      "betaVersionNumber": 590176,
      "compatibleVersionNumber": 524434,
      "productionReleaseSHA1": "f75b73c44bc3b8f94948e787e1d2c10a3a62bc5f",
      "betaReleaseSHA1": "7dbc928274024c0fec66afb2cd461351ab8a9c72",
      "compatibleReleaseSHA1": "b897e81655879cc0cf7a73fad79aa615cb17da31",
      "productionReleaseFileLength": 201996782,
      "betaReleaseFileLength": 205755882,
      "compatibleReleaseFileLength": 178140054,
      "existingFWContentID": ""
    },
    "Sebring": {
      "firmwareUpdateSource": null,
      "firmwareUpdateSourceFilePath": "",
      "firmwareUpdateSourceUrl": "",
      "firmwareUpdateStandardTargetFileName": "sebring-update.bsfw",
      "firmwareUpdateDifferentTargetFileName": "sebring-update_different.bsfw",
      "firmwareUpdateNewerTargetFileName": "sebring-update_newer.bsfw",
      "firmwareUpdateSaveTargetFileName": "sebring-update_save.bsfw",
      "firmwareUpdateVersion": "",
      "productionReleaseURL": "https://firmware.bsn.cloud/sebring-8.5.64-update.bsfw",
      "betaReleaseURL": "https://firmware.bsn.cloud/sebring-9.1.96-update.bsfw",
      "compatibleReleaseURL": "https://firmware.bsn.cloud/sebring-8.2.17.3-update.bsfw",
      "productionVersion": "8.5.64",
      "betaVersion": "9.1.96",
      "compatibleVersion": "8.2.17.3",
      "productionVersionNumber": 525632,
      "betaVersionNumber": 590176,
      "compatibleVersionNumber": 524817,
      "productionReleaseSHA1": "122a5cdf8315e3dbd63109fe1aa5c5a914be73d9",
      "betaReleaseSHA1": "98fb4b6d5574202223a6cbfad7c4b194d418995a",
      "compatibleReleaseSHA1": "399b5dd4203a8283f8e357a41e0168c33eb2418a",
      "productionReleaseFileLength": 80272950,
      "betaReleaseFileLength": 79840016,
      "compatibleReleaseFileLength": 74604516,
      "existingFWContentID": ""
    },
    "Raptor": {
      "firmwareUpdateSource": null,
      "firmwareUpdateSourceFilePath": "",
      "firmwareUpdateSourceUrl": "",
      "firmwareUpdateStandardTargetFileName": "raptor-update.bsfw",
      "firmwareUpdateDifferentTargetFileName": "raptor-update_different.bsfw",
      "firmwareUpdateNewerTargetFileName": "raptor-update_newer.bsfw",
      "firmwareUpdateSaveTargetFileName": "raptor-update_save.bsfw",
      "firmwareUpdateVersion": "",
      "productionReleaseURL": "https://firmware.bsn.cloud/raptor-9.1.92.1-update.bsfw",
      "betaReleaseURL": "https://firmware.bsn.cloud/raptor-9.1.96-update.bsfw",
      "compatibleReleaseURL": "https://firmware.bsn.cloud/raptor-9.0.22.3-update.bsfw",
      "productionVersion": "9.1.92.1",
      "betaVersion": "9.1.96",
      "compatibleVersion": "9.0.22.3",
      "productionVersionNumber": 590172,
      "betaVersionNumber": 590176,
      "compatibleVersionNumber": 589846,
      "productionReleaseSHA1": "e3400e17e766b13494a4e378bb47c2176ddb42eb",
      "betaReleaseSHA1": "c13c2ddd7a3f055aeac696ba96f2db2049802c83",
      "compatibleReleaseSHA1": "245ef90811a7bcd2e1ca862b451c4d8abfaa5526",
      "productionReleaseFileLength": 406106406,
      "betaReleaseFileLength": 406185446,
      "compatibleReleaseFileLength": 343723720,
      "existingFWContentID": ""
    },
    "Cobra": {
      "firmwareUpdateSource": null,
      "firmwareUpdateSourceFilePath": "",
      "firmwareUpdateSourceUrl": "",
      "firmwareUpdateStandardTargetFileName": "cobra-update.bsfw",
      "firmwareUpdateDifferentTargetFileName": "cobra-update_different.bsfw",
      "firmwareUpdateNewerTargetFileName": "cobra-update_newer.bsfw",
      "firmwareUpdateSaveTargetFileName": "cobra-update_save.bsfw",
      "firmwareUpdateVersion": "",
      "productionReleaseURL": "https://firmware.bsn.cloud/cobra-9.1.92.1-update.bsfw",
      "betaReleaseURL": "https://firmware.bsn.cloud/cobra-9.1.96-update.bsfw",
      "compatibleReleaseURL": "https://firmware.bsn.cloud/cobra-9.0.75-update.bsfw",
      "productionVersion": "9.1.92.1",
      "betaVersion": "9.1.96",
      "compatibleVersion": "9.0.75",
      "productionVersionNumber": 590172,
      "betaVersionNumber": 590176,
      "compatibleVersionNumber": 589899,
      "productionReleaseSHA1": "61c9f4ed254b9ad13e6274f3e2fc8e5a9bdec34e",
      "betaReleaseSHA1": "edb04f5839c920ce0d6773d2136205d5b6a17dee",
      "compatibleReleaseSHA1": "a213680fa5d8fb9437bee43888c37b9fd932f62d",
      "productionReleaseFileLength": 447185234,
      "betaReleaseFileLength": 447276148,
      "compatibleReleaseFileLength": 259763696,
      "existingFWContentID": ""
    },
    "Camaro": {
      "firmwareUpdateSource": null,
      "firmwareUpdateSourceFilePath": "",
      "firmwareUpdateSourceUrl": "",
      "firmwareUpdateStandardTargetFileName": "camaro-update.bsfw",
      "firmwareUpdateDifferentTargetFileName": "camaro-update_different.bsfw",
      "firmwareUpdateNewerTargetFileName": "camaro-update_newer.bsfw",
      "firmwareUpdateSaveTargetFileName": "camaro-update_save.bsfw",
      "firmwareUpdateVersion": "",
      "productionReleaseURL": "https://firmware.bsn.cloud/cobra-9.1.92.1-update.bsfw",
      "betaReleaseURL": "https://firmware.bsn.cloud/cobra-9.1.96-update.bsfw",
      "compatibleReleaseURL": "https://firmware.bsn.cloud/cobra-9.1.32-update.bsfw",
      "productionVersion": "9.1.92.1",
      "betaVersion": "9.1.96",
      "compatibleVersion": "9.1.32",
      "productionVersionNumber": 590172,
      "betaVersionNumber": 590176,
      "compatibleVersionNumber": 590112,
      "productionReleaseSHA1": "61c9f4ed254b9ad13e6274f3e2fc8e5a9bdec34e",
      "betaReleaseSHA1": "edb04f5839c920ce0d6773d2136205d5b6a17dee",
      "compatibleReleaseSHA1": "f220484a63e9e84b1bc91860eeea2cb61719b61e",
      "productionReleaseFileLength": 447185234,
      "betaReleaseFileLength": 447276148,
      "compatibleReleaseFileLength": 400990236,
      "existingFWContentID": ""
    },
    "Thor": {
      "firmwareUpdateSource": null,
      "firmwareUpdateSourceFilePath": "",
      "firmwareUpdateSourceUrl": "",
      "firmwareUpdateStandardTargetFileName": "thor-update.bsfw",
      "firmwareUpdateDifferentTargetFileName": "thor-update_different.bsfw",
      "firmwareUpdateNewerTargetFileName": "thor-update_newer.bsfw",
      "firmwareUpdateSaveTargetFileName": "thor-update_save.bsfw",
      "firmwareUpdateVersion": "",
      "productionReleaseURL": "https://firmware.bsn.cloud/thor-9.1.92.1-update.bsfw",
      "betaReleaseURL": "https://firmware.bsn.cloud/thor-9.1.96-update.bsfw",
      "compatibleReleaseURL": "https://firmware.bsn.cloud/thor-9.1.32-update.bsfw",
      "productionVersion": "9.1.92.1",
      "betaVersion": "9.1.96",
      "compatibleVersion": "9.1.32",
      "productionVersionNumber": 590172,
      "betaVersionNumber": 590176,
      "compatibleVersionNumber": 590112,
      "productionReleaseSHA1": "abb1f11ffd355a34c4b6a7b009b7924f6323a3ab",
      "betaReleaseSHA1": "4ae356ee5626914f193c6f21fc3e2432150da569",
      "compatibleReleaseSHA1": "bb88b23d08810424a2fef397d82e00d8fb5c1751",
      "productionReleaseFileLength": 328117114,
      "betaReleaseFileLength": 328398796,
      "compatibleReleaseFileLength": 287594196,
      "existingFWContentID": ""
    }
  },
  "firmwareUpdateType": "standard",
  "setupType": "lfn",
  "bsnDeviceRegistrationTokenEntity": {
    "token": "{{.RegistrationToken}}",
    "scope": "cert",
    "validFrom": "{{.TokenValidFrom}}",
    "validTo": "{{.TokenValidTo}}"
  },
  "enableSerialDebugging": true,
  "enableSystemLogDebugging": true,
  "remoteDwsEnabled": true,
  "dwsEnabled": false,
  "dwsPasswordEdited": false,
  "dwsPassword": "",
  "dwsPasswordPreviousSavedTimeStamp": 0,
  "lwsEnabled": false,
  "lwsConfig": "status",
  "lwsUserName": "",
  "lwsUsernameEdited": false,
  "lwsPassword": "",
  "lwsPasswordEdited": false,
  "lwsEnableUpdateNotifications": true,
  "bsnCloudEnabled": true,
  "deviceName": "{{.DeviceName}}",
  "deviceDescription": "{{.DeviceDescription}}",
  "unitNamingMethod": "appendUnitIDToUnitName",
  "timeZone": "PST",
  "bsnGroupName": "{{.GroupName}}",
  "timeBetweenNetConnects": 300,
  "sfnWebFolderUrl": "",
  "sfnUserName": "",
  "sfnPassword": "",
  "sfnEnableBasicAuthentication": false,
  "playbackLoggingEnabled": true,
  "eventLoggingEnabled": true,
  "diagnosticLoggingEnabled": true,
  "stateLoggingEnabled": true,
  "variableLoggingEnabled": true,
  "uploadLogFilesAtBoot": false,
  "uploadLogFilesAtSpecificTime": false,
  "uploadLogFilesTime": 0,
  "logHandlerUrl": "",
  "enableRemoteSnapshot": false,
  "remoteSnapshotInterval": 15,
  "remoteSnapshotMaxImages": 5,
  "remoteSnapshotJpegQualityLevel": 50,
  "remoteSnapshotScreenOrientation": "Landscape",
  "remoteSnapshotHandlerUrl": "",
  "idleScreenColor": {
    "r": 0,
    "g": 0,
    "b": 0,
    "a": 1
  },
  "networkDiagnosticsEnabled": false,
  "testEthernetEnabled": false,
  "testWirelessEnabled": false,
  "testInternetEnabled": false,
  "useCustomSplashScreen": false,
  "BrightWallName": "",
  "BrightWallScreenNumber": "",
  "contentDownloadsRestricted": false,
  "contentDownloadRangeStart": 0,
  "contentDownloadRangeEnd": 0,
  "usbUpdatePassword": "",
  "inheritNetworkProperties": true,
  "internalCaArtifacts": [],
  "network": {
    "timeServers": [
      "http://time.brightsignnetwork.com"
    ],
    "hostname": null,
    "dns": null,
    "proxyServer": null,
    "proxyBypass": null,
    "interfaces": [
      {
        "id": "wired_eth0",
        "name": "eth0",
        "type": "Ethernet",
        "proto": "DHCPv4",
        "ip": [],
        "gateway": null,
        "dns": [],
        "showInUi": true,
        "rateLimitDuringInitialDownloads": null,
        "rateLimitInsideContentDownloadWindow": null,
        "rateLimitOutsideContentDownloadWindow": null,
        "contentDownloadEnabled": true,
        "textFeedsDownloadEnabled": true,
        "mediaFeedsDownloadEnabled": true,
        "healthReportingEnabled": true,
        "logsUploadEnabled": true,
        "wpaSettings": {
          "enableWPAEnterpriseAuthentication": false,
          "wpaEnterpriseVariant": "WPAEnterpriseEapTls",
          "eapCertificateType": "WPAEapTlsPKCS",
          "eapCertificateFile": null,
          "eapCertificatePassphrase": "",
          "eapPemOrDerKeyFile": null,
          "peapUsername": "",
          "peapPassphrase": "",
          "caCertificateFile": null
        }
      }
    ],
    "certificateName": "",
    "certificateFile": null
  }
}
```
