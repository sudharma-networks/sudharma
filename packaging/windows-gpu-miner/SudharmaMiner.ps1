# Sudharma one-click GPU miner launcher for Windows.
# Double-click this file if Start Mining.bat is blocked by your browser.

$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot

Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing

$form = New-Object System.Windows.Forms.Form
$form.Text = 'Sudharma GPU Miner'
$form.Size = New-Object System.Drawing.Size(520, 320)
$form.StartPosition = 'CenterScreen'
$form.FormBorderStyle = 'FixedDialog'
$form.MaximizeBox = $false

$label = New-Object System.Windows.Forms.Label
$label.Text = 'Paste your Sudharma wallet address. Mining connects to public-testnet automatically.'
$label.AutoSize = $false
$label.Size = New-Object System.Drawing.Size(480, 40)
$label.Location = New-Object System.Drawing.Point(16, 16)
$form.Controls.Add($label)

$addressBox = New-Object System.Windows.Forms.TextBox
$addressBox.Size = New-Object System.Drawing.Size(480, 24)
$addressBox.Location = New-Object System.Drawing.Point(16, 64)
$form.Controls.Add($addressBox)

$savedPath = Join-Path $env:LOCALAPPDATA 'Sudharma\gpu-miner\reward-address.txt'
if (Test-Path $savedPath) {
  $addressBox.Text = (Get-Content -Raw $savedPath).Trim()
}

$start = New-Object System.Windows.Forms.Button
$start.Text = 'Start Mining'
$start.Size = New-Object System.Drawing.Size(120, 32)
$start.Location = New-Object System.Drawing.Point(16, 104)
$form.Controls.Add($start)

$log = New-Object System.Windows.Forms.TextBox
$log.Multiline = $true
$log.ReadOnly = $true
$log.ScrollBars = 'Vertical'
$log.Size = New-Object System.Drawing.Size(480, 140)
$log.Location = New-Object System.Drawing.Point(16, 148)
$form.Controls.Add($log)

$minerExe = Join-Path $PSScriptRoot 'sudharma-miner.exe'
if (-not (Test-Path $minerExe)) {
  [System.Windows.Forms.MessageBox]::Show('sudharma-miner.exe was not found in this folder.', 'Sudharma GPU Miner')
  exit 1
}

$start.Add_Click({
  $address = $addressBox.Text.Trim()
  if ($address.Length -ne 40) {
    [System.Windows.Forms.MessageBox]::Show('Enter your 40-character Sudharma wallet address.', 'Sudharma GPU Miner')
    return
  }
  $start.Enabled = $false
  $log.Clear()
  $log.AppendText("Starting Sudharma GPU miner...`r`n")
  $psi = New-Object System.Diagnostics.ProcessStartInfo
  $psi.FileName = $minerExe
  $psi.Arguments = "--address $address"
  $psi.WorkingDirectory = $PSScriptRoot
  $psi.UseShellExecute = $false
  $psi.RedirectStandardOutput = $true
  $psi.RedirectStandardError = $true
  $psi.CreateNoWindow = $true
  $proc = New-Object System.Diagnostics.Process
  $proc.StartInfo = $psi
  [void]$proc.Start()
  while (-not $proc.StandardOutput.EndOfStream) {
    $line = $proc.StandardOutput.ReadLine()
    if ($line) { $log.AppendText("$line`r`n") }
  }
  while (-not $proc.StandardError.EndOfStream) {
    $line = $proc.StandardError.ReadLine()
    if ($line) { $log.AppendText("$line`r`n") }
  }
  $proc.WaitForExit()
  $start.Enabled = $true
})

[void]$form.ShowDialog()
