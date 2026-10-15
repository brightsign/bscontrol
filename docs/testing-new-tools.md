# Testing Modified bdeploy Tools

All commands assume `BS_CLIENT_ID`, `BS_SECRET`, and `BS_NETWORK` are already set. Replace `"some-package"` and `SOME_SERIAL` with real values from your network.

## 1. Verify New Flags Appear in Help

```bash
./bin/bsn-bdeploy-setup-add --help 2>&1 | grep -E "package-name|device-name|device-description|template"
./bin/bsn-bdeploy-setup-list --help 2>&1 | grep -E "package-name|package"
./bin/bsn-bdeploy-setup-get --help 2>&1 | grep "package-name"
./bin/bsn-bdeploy-setup-update --help 2>&1 | grep "package-name"
./bin/bsn-bdeploy-setup-delete --help 2>&1 | grep "package-name"
./bin/bsn-bdeploy-device-associate --help 2>&1 | grep -E "package-name|device-name|device-description"
```

## 2. Test Environment Variable Resolution

### `bdeploy-list-setups` -- filter by `BS_PACKAGE_NAME`

```bash
# Via flag (existing behavior, renamed)
./bin/bsn-bdeploy-setup-list --package-name "some-package"

# Via old alias (backward compat)
./bin/bsn-bdeploy-setup-list --package "some-package"

# Via env var (new)
export BS_PACKAGE_NAME="some-package"
./bin/bsn-bdeploy-setup-list
unset BS_PACKAGE_NAME
```

All three should produce the same filtered output.

### `bdeploy-get-setup` -- lookup by `BS_PACKAGE_NAME`

```bash
# Via new alias
./bin/bsn-bdeploy-setup-get --package-name "some-package"

# Via old flag (backward compat)
./bin/bsn-bdeploy-setup-get --setup-name "some-package"

# Via env var
BS_PACKAGE_NAME="some-package" ./bin/bsn-bdeploy-setup-get
```

### `bdeploy-delete-setup` -- lookup by `BS_PACKAGE_NAME`

```bash
# Dry run: just confirm it finds the setup (will prompt for confirmation)
./bin/bsn-bdeploy-setup-delete --package-name "some-package"

# Via env var
BS_PACKAGE_NAME="some-package" ./bin/bsn-bdeploy-setup-delete
```

## 3. Test `bdeploy-add-setup` Template Mode

### Template mode with flags

```bash
./bin/bsn-bdeploy-setup-add \
  --package-name "test-template-v1" \
  --device-name "Test Device" \
  --device-description "Created via template mode" \
  --group "Default" \
  --verbose
```

### Template mode with env vars

```bash
export BS_PACKAGE_NAME="test-env-v1"
export BS_DEVICE_NAME="Env Device"
export BS_DEVICE_DESCRIPTION="Created via env vars"
./bin/bsn-bdeploy-setup-add --verbose
unset BS_PACKAGE_NAME BS_DEVICE_NAME BS_DEVICE_DESCRIPTION
```

### Flag overrides env var

```bash
export BS_PACKAGE_NAME="should-be-overridden"
./bin/bsn-bdeploy-setup-add --package-name "flag-wins" --verbose
unset BS_PACKAGE_NAME
```

### Config file mode (backward compat)

```bash
./bin/bsn-bdeploy-setup-add examples/bsn-bdeploy-setup-add/config.json --verbose
```

### Error case: no package name and no config file

```bash
./bin/bsn-bdeploy-setup-add
# Should print error and usage
```

### Custom template path

```bash
./bin/bsn-bdeploy-setup-add \
  --template setups/setup-template.json \
  --package-name "custom-template-test" \
  --verbose
```

## 4. Test `bdeploy-update-setup` with `--package-name` Override

```bash
# Flag overrides config file's packageName
./bin/bsn-bdeploy-setup-update \
  --setup-name "existing-setup" \
  --package-name "renamed-package" \
  examples/bsn-bdeploy-setup-update/example-config.json \
  --verbose

# Env var overrides config file's packageName
BS_PACKAGE_NAME="env-renamed" ./bin/bsn-bdeploy-setup-update \
  --setup-name "existing-setup" \
  examples/bsn-bdeploy-setup-update/example-config.json
```

## 5. Test `bdeploy-associate` with Aliases

```bash
# New alias flags
./bin/bsn-bdeploy-device-associate \
  --serial SOME_SERIAL \
  --package-name "some-package" \
  --device-name "Lobby Display" \
  --device-description "Main entrance" \
  --create

# Env vars
export BS_PACKAGE_NAME="some-package"
export BS_DEVICE_NAME="Lobby Display"
export BS_DEVICE_DESCRIPTION="Main entrance"
./bin/bsn-bdeploy-device-associate --serial SOME_SERIAL --create
unset BS_PACKAGE_NAME BS_DEVICE_NAME BS_DEVICE_DESCRIPTION

# Old flags still work
./bin/bsn-bdeploy-device-associate \
  --serial SOME_SERIAL \
  --setup-name "some-package" \
  --name "Lobby Display" \
  --description "Main entrance" \
  --create
```

## 6. Cleanup Test Records

After testing, remove any setup records created during testing:

```bash
./bin/bsn-bdeploy-setup-delete --package-name "test-template-v1" --force
./bin/bsn-bdeploy-setup-delete --package-name "test-env-v1" --force
./bin/bsn-bdeploy-setup-delete --package-name "flag-wins" --force
./bin/bsn-bdeploy-setup-delete --package-name "custom-template-test" --force
```
