# Windows installer smoke test, run by CI and before every release.
# It installs into a temp data folder and a throwaway HKCU key (never the real
# ones), re-installs while the helper is running (as when Chrome has YouTube
# Music open), then uninstalls.
$ErrorActionPreference = 'Stop'
if (-not $env:YTM_DISCORD_DATA_DIR -or -not $env:YTM_DISCORD_REGISTRY_KEY) {
  throw 'set YTM_DISCORD_DATA_DIR and YTM_DISCORD_REGISTRY_KEY to throwaway locations first'
}
$origin = 'chrome-extension://bfcijjaglbmlpnnpnekjcamlmgdkdebb/'
$root = "HKCU:\$env:YTM_DISCORD_REGISTRY_KEY"

go build -o ytm-discord.exe .
if ($LASTEXITCODE -ne 0) { throw 'build failed' }

# 1. Install
./ytm-discord.exe 123456789012345678
if ($LASTEXITCODE -ne 0) { throw 'install failed' }
$key = "$root\Google\Chrome\NativeMessagingHosts\local.ytm_discord_presence"
$manifestFile = (Get-ItemProperty $key).'(default)'
if (-not (Test-Path $manifestFile)) { throw "host manifest missing: $manifestFile" }
$manifest = Get-Content $manifestFile -Raw | ConvertFrom-Json
$helperExe = $manifest.path
if (-not (Test-Path $helperExe)) { throw "helper missing: $helperExe" }
if ($manifest.allowed_origins[0] -ne $origin) { throw "allowed_origins: $($manifest.allowed_origins)" }
& $helperExe --version
if ($LASTEXITCODE -ne 0) { throw 'the installed helper does not run' }
if (-not (Test-Path "$env:YTM_DISCORD_DATA_DIR\extension\manifest.json")) { throw 'extension not unpacked' }
if (-not (Test-Path "$root\Microsoft\Edge\NativeMessagingHosts\local.ytm_discord_presence")) { throw 'Edge not registered' }

# 2. Re-install while the helper runs: the running exe must be renamed aside.
$psi = [System.Diagnostics.ProcessStartInfo]::new($helperExe, $origin)
$psi.UseShellExecute = $false
$psi.RedirectStandardInput = $true
$psi.RedirectStandardOutput = $true
$helper = [System.Diagnostics.Process]::Start($psi)
Start-Sleep -Seconds 1
if ($helper.HasExited) { throw "the helper exited early (code $($helper.ExitCode))" }
./ytm-discord.exe 123456789012345678
if ($LASTEXITCODE -ne 0) { throw 're-install with the helper running failed' }
$olds = @(Get-ChildItem "$helperExe.old-*" -ErrorAction SilentlyContinue)
if ($olds.Count -ne 1) { throw "expected one renamed old helper, found $($olds.Count)" }
& $helperExe --version
if ($LASTEXITCODE -ne 0) { throw 'the replaced helper does not run' }
$helper.StandardInput.Close() # like Chrome closing the tab
if (-not $helper.WaitForExit(10000)) { $helper.Kill(); throw 'the helper did not exit when stdin closed' }

# 3. The next install cleans up the renamed copy.
./ytm-discord.exe 123456789012345678
if ($LASTEXITCODE -ne 0) { throw 'third install failed' }
if (Get-ChildItem "$helperExe.old-*" -ErrorAction SilentlyContinue) { throw 'the old helper was not cleaned up' }

# 4. Uninstall
./ytm-discord.exe --uninstall
if ($LASTEXITCODE -ne 0) { throw 'uninstall failed' }
if (Test-Path $key) { throw 'registry key not removed' }
if (Test-Path $env:YTM_DISCORD_DATA_DIR) { throw 'data folder not removed' }
Remove-Item -Recurse $root
Write-Host 'smoke test passed'
