# sf-Compatible Mode Checklist

Reference: `sf` 2.138.6. Update this file as each item is finished.

## Step 1: Dispatch And Output
- [x] Binary name detection in `main.go`
- [x] sf root command, colon-form ids, legacy command aliases
- [x] Legacy flag names (`--targetusername`, `-u`, `--targetdevhubusername`, `--apiversion`)
- [x] `--json` envelope, error envelope, human error format, table renderer
- [x] `ErrorAndExit` handler hook
- [x] `version`

## Step 2: sf State
- [x] Config read/write (`~/.sf/config.json`, `<project>/.sf/config.json`, env)
- [x] Alias read/write (`~/.sfdx/alias.json`)
- [x] Auth file write and delete, field encryption
- [x] Org resolution (flag, env, local config, global config)
- [x] Save refreshed tokens to the sf auth file
- [x] `alias list`, `alias set`, `alias unset`
- [x] `config get`, `config list`, `config set`, `config unset`
- [x] `org list`, `org list auth`, `org display`, `org logout`

## Step 3: org
- [x] `org login web` (not exercised against a live org yet)
- [x] `org login jwt` (not exercised against a live org yet)
- [x] `org login access-token`
- [x] `org open` (format taken from the sf plugin source; the local test server has no single-access endpoint)
- [x] `org display user`
- [ ] `org create scratch` (including definition file)
- [x] `org list limits` (format taken from the sf plugin source; the local test server has no limits resource)
- [x] `org list metadata`
- [x] `org list metadata-types`

## Step 4: data
- [x] `data query`
- [ ] `data search`
- [x] `data get record`, `data create record`, `data update record`, `data delete record`
- [ ] `data import bulk`, `data update bulk`, `data upsert bulk`, `data delete bulk`, `data export bulk`
- [ ] `data import resume`, `data update resume`, `data upsert resume`, `data delete resume`, `data export resume`
- [ ] `data bulk results`
- [ ] `data resume`
- [ ] `force data bulk delete`, `force data bulk upsert`, `force data bulk status`

## Step 5: apex, sobject, api, template
- [ ] `apex run`
- [ ] `apex run test`
- [ ] `apex get log`, `apex list log`, `apex tail log`
- [ ] `apex trace create`, `apex trace list`, `apex trace delete`
- [ ] `sobject describe`, `sobject list`
- [ ] `api request rest`
- [ ] `template generate apex class`, `template generate apex trigger`
- [ ] `template generate visualforce page`, `template generate visualforce component`

## Step 6: project
- [ ] Move deploy and fetch helpers from `command/` to `lib`
- [ ] `project deploy start`, `project deploy validate`
- [ ] `project deploy quick`, `project deploy report`, `project deploy resume`, `project deploy cancel`
- [ ] `project retrieve start`
- [ ] `project convert source`, `project convert mdapi`

## Step 7: package
- [ ] `package create`, `package list`
- [ ] `package install`, `package installed list`, `package uninstall`
- [ ] `package version create`, `package version list`, `package version report`, `package version promote`

## Step 8: Documentation And Build
- [ ] `docs/sf-compatibility.md` (mapping both directions, missing commands)
- [ ] Generated `docs/sf/` from the sf command tree
- [ ] README section
- [ ] Makefile `sf` and `install-sf` targets
