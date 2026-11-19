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
- [ ] Config read/write (`~/.sf/config.json`, `<project>/.sf/config.json`, env)
- [ ] Alias read/write (`~/.sfdx/alias.json`)
- [ ] Auth file write and delete, field encryption
- [ ] Org resolution (flag, env, local config, global config)
- [ ] Save refreshed tokens to the sf auth file
- [ ] `alias list`, `alias set`, `alias unset`
- [ ] `config get`, `config list`, `config set`, `config unset`
- [ ] `org list`, `org list auth`, `org display`, `org logout`

## Step 3: org
- [ ] `org login web`
- [ ] `org login jwt`
- [ ] `org login access-token`
- [ ] `org open`
- [ ] `org display user`
- [ ] `org create scratch` (including definition file)
- [ ] `org list limits`
- [ ] `org list metadata`
- [ ] `org list metadata-types`

## Step 4: data
- [ ] `data query`
- [ ] `data search`
- [ ] `data get record`, `data create record`, `data update record`, `data delete record`
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
