<!--
DariTerm help (the [Help] window next to Settings, D155). To change the text, edit only this file and ko.md.
Format: one "## Title {#id}" is one topic. The id must be the same in every language (used by the Settings help links).
Available formatting: ### subheading, paragraph, "- " list, "1. " numbered list, table (| a | b |), "> " note, **bold**, `code`, ``` code block.
A "Q: question" line and the answer lines right below it (up to a blank line) make one question card (FAQ).
A "![caption](file.png)" line is a picture: locales/help/img/<lang>/file (falls back to en → ko; skipped if missing). Retake pictures with `npm run help:capture` in frontend.
-->

## First launch {#start}

DariTerm is a terminal for people who work on servers. It runs SSH, Telnet, serial and local terminals in tabs and split panes.

1. **Language** — pick the display language on first launch. You can change it later in Settings > Appearance.
2. **Usage** — choose personal or business use.
3. **App lock password** — encrypts saved server passwords and keys. If you forget it, saved passwords cannot be recovered.
4. **Add a server** — use [Add server] in the sidebar, or import from another program.

> All data stays on this PC. Use git sync or backup to share it with other PCs.

## Around the window {#screen}

- **Sidebar**: server list (folders, favorites, recent) and search. The icon group at the top left has the server list, macros & snippets, settings and help buttons.
- **Tab bar**: one tab per connection. A tab can be split into panes.
- **Pane header**: server name and target. If port forwarding is set, a `⇄ N` button appears. The `↑1.2 K ↓34 M` before the buttons on the right is this session's **traffic** (↑ sent = input, ↓ received = output, since the session opened), updated every second. It is hidden when the pane is narrow.
- **Status bar**: connection state and time of the current pane, usage type, macro recording badge, and "Unlocked" at the right end (click to lock now).

![Main window — server list and macros & snippets in the sidebar](screen.png)

## Adding servers {#servers}

1. **[Add server]** in the sidebar (or right-click a folder > Add server here).
2. **General** tab: display name, protocol (SSH, Telnet, serial, local), address and port, folder, color and environment (production, development, ...).
3. **Authentication** tab (SSH): password, key file, SSH agent, or ask every time.
   - Passwords and keys are stored encrypted in the vault.
   - Servers that use the same account can share one "shared account".
4. **Advanced** tab: jump server, proxy, agent forwarding, keepalive, encoding, command after connect, keep session (tmux).
5. **Port forwarding** tab (SSH), **Automation** tab (login script and this server's output triggers).
6. **[Save]** or **[Save and connect]**.

![Server editor > General tab](servers.png)

Drag servers to move them between folders, or select several with Ctrl/Shift to move or delete them together. Click the Favorites and Recent headings to collapse them.

## Connecting {#connect}

- **Double-click** a server to connect in a new tab.
- **Ctrl+K** (quick connect): type a few letters of the server name and press Enter. Type `user@host:port` to connect to a server you have not saved. Start with `>` for the command list.
- **Ctrl+Alt+T**: a terminal on this PC.
- If the connection drops, Settings > Connection "Reconnect when disconnected" retries after 2, 4, 8, 16 and 30 seconds. After it stops, press Enter in that pane to reconnect. The reason for the disconnect is shown in the terminal.

![Quick connect (Ctrl+K)](connect.png)

## Login and host keys {#auth}

- A **server you connect to for the first time** shows its key fingerprint. If it matches what the server administrator told you, choose [Trust and connect]. If the key changes later you get a warning — do not connect unless you know why.
- When asked for a password, turn on [Save] so you are not asked again.
- Key files can be OpenSSH or PuTTY keys (`.ppk`, versions 2 and 3). Encrypted keys ask for the passphrase once.
- If the server asks extra questions (OTP and so on), an input box appears each time.

## Vault and app lock {#vault}

Server passwords, keys, and jump server and proxy accounts are stored in the **vault**, encrypted with the app lock password.

![Settings > Security — auto-lock and saved accounts](vault.png)

- **Auto-lock**: Settings > Security > Auto-lock (minutes). The app locks after that long without input, when **Windows is locked**, or on sleep. Connections stay open. 0 turns it off.
- Windows locking is judged by the **lock state of the Windows session**. So the app does not lock while only a screen saver is showing and Windows is not actually locked (a screen saver with "On resume, display logon screen" turned off). The idle time rule still applies.
- **Unlock with Windows sign-in**: turn it on in Settings > Security to open with your Windows account instead of the password.
- **Saved accounts**: view, rename or delete saved passwords and keys in Settings > Security.
- Changing the lock password keeps everything you saved.

## Using the terminal {#terminal}

- **Copy**: selecting with the mouse copies right away (can be turned off in Settings > Input). **Paste**: Ctrl+V or right-click.
- **Find**: Ctrl+Shift+F. Move up and down through the matches.
- **Font size**: Ctrl+= / Ctrl+- / Ctrl+0.
- Pasting an image uploads it to a temporary folder on the server and types its path.
- On a **production server** (environment "Production"), pasting a dangerous command (`rm -rf` and so on) or pressing Enter on one asks once more. Edit the patterns in Settings > Input.

## Tabs and split panes {#tabs}

| What you want | How |
|---|---|
| Same server on the right | Ctrl+Shift+D |
| Same server below | Ctrl+Shift+E |
| Maximize one pane | Ctrl+Shift+Enter |
| Move between panes | Alt+arrow keys |
| Broadcast input (same input to all panes of the tab) | Ctrl+Shift+B |
| Move a tab | Ctrl+Shift+← / → or drag |

- A tab with several panes shows a pane count badge.
- Turn on Settings > Appearance "Restore last layout" to bring back tabs and panes next time.

## Jump servers {#jump}

Servers behind a firewall are reached through a jump server. Pick a saved server or type one in Server editor > Advanced > **Jump server**.

- If the jump server has its own jump server, the hops are passed in order.
- While connecting, a "Connecting through jump server ..." line appears for each hop.
- Importing an ssh config keeps `ProxyJump a,b` as hops (see Importing).

## Proxies and proxy commands {#proxy}

Choose it in Server editor > Advanced > **Proxy**. With a jump server, only the proxy of the outermost server (the one this PC connects to directly) is used.

![Server editor > Advanced — proxy command](proxy.png)

- **HTTP proxy (CONNECT)**, **SOCKS5 proxy**: address and port, and a user name and password if needed (saved in the vault as `{display name} proxy`).
- **Proxy command (ProxyCommand)**: uses the input and output of an external command as the path to the server, like OpenSSH's ProxyCommand.
  - `%h` becomes the server address, `%p` the port, `%r` the user name, and `%%` a `%`.
  - It runs as in the command prompt (cmd), so the program must be installed on this PC.
  - If there is no response within 20 seconds the command is stopped. On failure, the end of the command's error output is shown.

```
ssh -W %h:%p bastion
ncat --proxy proxy.example.com:8080 %h %p
```

## Port forwarding {#forward}

Add forwards in the Server editor > **Port forwarding** tab.

![Server editor > Port forwarding tab](forward.png)

| Type | Meaning |
|---|---|
| L (local) | Connections to a port on this PC go to an address on the server side |
| R (remote) | Connections to a port on the server go to an address on this PC's side |
| D (dynamic) | Opens a port on this PC as a SOCKS proxy |

- After connecting, the terminal shows one line per forward. Failed lines are red.
- Click the `⇄ N` button in the pane header (or right-click the pane > View port forwarding) for a table of type, listen address, destination and status. The status is "Active", with the number of open connections if there are any.
- **Traffic**: bytes sent (↑) and received (↓) per forward since connecting. [Refresh] reads them again.

## SSH agent and key files {#agent}

- **SSH agent**: with authentication set to "SSH agent", keys from the Windows OpenSSH agent and a running Pageant are both tried. Outside Windows, `SSH_AUTH_SOCK` is used.
- **Agent forwarding** (`ssh -A`): toggle in Server editor > Advanced (off by default). Lets `ssh` and `git` on the server use this PC's agent keys. Turn it on only for trusted servers.
- Key files can be OpenSSH or PuTTY `.ppk`, without conversion.

## Create an SSH key and add it to a server {#ssh-keys}

How to switch a server from password to key sign-in.

1. **Settings > Security > Saved accounts** > [Create SSH key]: enter a name, the type (Ed25519 recommended, RSA 4096 for older servers) and an optional passphrase, then [Create]. The key is stored encrypted in the vault.
2. In a pane signed in to the server with a password, right-click > **Add public key to this server…**, pick the key and click [Add]. One line is added to `~/.ssh/authorized_keys` on the server (nothing changes if it is already there).
3. Leave "Sign in to this server with this key from next time" on to switch the server settings to key sign-in as well.

- [Public key] on a key account shows the public key line and fingerprint to copy (for other servers, GitHub and so on).
- Adding runs `sh` on the server. If the server has no `sh` (for example Windows OpenSSH), copy the public key and add it yourself.

## Keeping sessions (tmux) {#tmux}

Turn on Server editor > Advanced > **Keep session (tmux)** to work inside a tmux session on the server. Work keeps running when the pane closes or the connection drops, and reconnecting returns to that screen.

- If the session name is empty, the server name is used.
- tmux must be installed on the server. For this PC's terminal it works only in a WSL shell.

## Serial connections {#serial}

Connect to device consoles (routers, switches) over USB-serial. Set the protocol to **Serial** in the server editor and pick the port (COM3 and so on).

- Speed (default 9600), data bits, parity, stop bits, flow control (none, RTS/CTS, XON/XOFF), what Enter sends, local echo.
- Input the device has not accepted for over 2 seconds is dropped with a notice.

## Serial hex view, hex send and XMODEM {#serial-tools}

In the right-click menu of a serial pane (while connected).

- **Show as hex**: received bytes are shown as hex and text, like `01 41 FF  |.A.|` (16 bytes per line, each received chunk on a new line). Click again for "Show as text".
- **Send hex…**: sends bytes written like `01 03 00 00 00 0A C5 CD` as they are (spaced, run together or `0x01,0x03`). The window stays open so you can send the same value again.
- **Send with XMODEM / XMODEM-1K**: pick a file, then start receiving on the device (for example `loadx` in a boot loader). 1K uses 1024-byte blocks and is faster (only when the device uses CRC).
- **Receive with XMODEM**: pick a name to save as, then start sending on the device. The padding (0x1A) at the end of the last block is removed.
- During a transfer, keys are not sent to the device. Press **Ctrl+C** or "Cancel XMODEM transfer" to stop.

## File transfer (SFTP) {#sftp}

In an SSH pane, **Ctrl+Shift+S** opens the SFTP window.

- Browse server folders, drag files in to upload and out to download.
- Download folder, temp folder and cleanup of uploaded temp files are in Settings > File transfer.
- You can sync SFTP to the terminal's current folder, or `cd` the terminal to the SFTP folder.

## Sending files in the terminal (ZMODEM) {#zmodem}

In SSH and serial terminals, receive with `sz file` on the server and send with `rz`.

- Download folder and on/off are in Settings > File transfer.
- Ctrl+C cancels a download. If the server does not respond for 60 seconds, only the transfer ends (the connection stays).
- Transfers do not work inside tmux or screen. If it hangs, press Ctrl+X five times.

## Snippets and variables {#snippets}

Save frequent commands in **Macros & snippets** (the `<>` icon at the top left) and click to type them. Commands in the list have a green **Command** badge.

- [New snippet]: name, command (multi-line allowed), add Enter.
- In a tab with broadcast input on, it is typed into every pane of the tab.
- **Variables**: write `{{name}}` or `{{name:default}}` in a command; clicking it asks for the values and types the filled-in command.

```
systemctl restart {{service}} --no-block {{options:-q}}
```

> Something starting with a dot like `{{.Names}}`, or written as `\{{…}}`, is not a variable. Previous values are remembered only while the app is running.

## Macros {#macros}

A macro is a list of steps. It waits for the result of one input before sending the next. Macros have a **blue "Macro N steps"** badge in the list.

![Macros & snippets — the blue badge marks a macro](macros.png)

- **Create**: [New macro]. Each step has *wait for* text (empty: until output is quiet for 0.5 s, at most 10 s), *send* text and whether to press Enter. If the wait-for text does not appear within 30 seconds, the macro stops.
- **Record**: click [Record macro] (circle), type in the terminal as usual, and click again to stop. Every Enter becomes a step. A red badge shows in the status bar while recording.
- **Run**: click it in the list to run in the current pane. **In a tab with broadcast input on, it runs separately in every connected pane of the tab** (each pane waits for its own output). Panes already running a macro are skipped.
- **Shortcuts**: you can assign keys to "Start/stop macro recording", "Run last macro again" and "Stop macro in current pane" in Settings > Shortcuts (no default keys).
- Write control characters as `\x03` (Ctrl+C), `\e`, `\t`, `\r`, `\n`, `\\`.

> Macros are stored unencrypted. Do not type passwords while recording.

## Login script {#login-script}

Automates what you type right after connecting (`sudo su -`, a device's `enable`, ...). Server editor > **Automation** tab.

![Server editor > Automation — login script and this server's output triggers](login-script.png)

- Each step has *wait for* and *send* text. When the wait-for text appears in the server output, the send text + Enter is sent and the next step starts. An empty wait-for text sends immediately. Case is ignored.
- Runs once from the top on every connect (including reconnects). Stops if a step does not advance within 20 seconds or if you type yourself.
- `{{password}}` becomes the password used to log in to that server (SSH password login).
- **Vault value**: for passwords other than the login one (`su`, `enable`), pick a vault item in that step's [Vault value]. You don't write the password in the send text, and the file keeps only the item reference.
- "Command after connect" on the Advanced tab runs after the script finishes.

## Keyword highlighting {#highlight}

Settings > **Output rules** > Keyword highlighting. Shows words like `ERROR` or `FAIL` in the color you choose.

- Each rule: text to find (regular expression allowed), case, color, fill background.
- Only the display changes. Log files are not affected.

## Output triggers {#triggers}

When certain text appears in the output, show a notification, play a sound or reply automatically. Output of hidden tabs is watched too.

![Settings > Output rules — output triggers and Windows notifications](triggers.png)

- **Rules for all connections**: Settings > Output rules > Output triggers.
- **Rules for one server**: Server editor > Automation > "Output triggers for this server". Used only in panes connected to that server, together with the Settings rules. Saved with the server.
- Each rule: text to find (regular expression allowed), case, **notify**, **sound**, **auto-reply** (text to send, Enter).
- **Windows notifications**: when the app is in the background, notifications also go to the Windows notification center. Turn this off with "Also show a Windows notification when the app is in the background" in Settings > Output rules.
- The echo of a line you are typing and full-screen programs such as vim are not watched. The same rule runs at most once per second.
- Production servers get a notice instead of an auto-reply. If an auto-reply repeats more than 5 times in 10 seconds, that rule is turned off for that connection.

## Session logs {#log}

Turn it on in Settings > **Logs**.

![Settings > Log](log.png)

- Saves terminal output to a text file per connection (`{display name}-{date-time}.log`). [Open log folder] shows where.
- Timestamp each line, keep ANSI (colors), flush interval, retention (old files are deleted automatically).
- Turning it on also starts logging connections that are already open.

## Importing from other programs {#import}

Sidebar menu > **Import servers**.

- **PuTTY** (registry), **SecureCRT** (session folder), **Xshell** (export file `.xts`, session file `.xsh`), **ssh config** (`~/.ssh/config`), **AWS EC2** (aws CLI).
- Folder structure, port forwards and key files are imported. Passwords are not.
- In ssh config, multi-hop **ProxyJump** (`ProxyJump a,b`) becomes a chain of jump servers, and **ProxyCommand** becomes the server's proxy command.
- SecureCRT login scripts (logon actions) are not imported yet; you get a notice.
- Candidates that look like servers you already have start unselected.

## Sync and backup {#sync}

- **git sync**: Settings > git sync. Pushes the server list and vault as encrypted files to your own git repository (private recommended) to keep PCs in step.
- **Backup**: Settings > Backup. Export to and import from a single file. When importing, merge with or replace the current list.
- Macros, snippets and output rules move along too.

## Keyboard shortcuts {#shortcuts}

In Settings > **Shortcuts**, change or clear the key of each command. Turning off "Use shortcuts" disables all app shortcuts so those keys go to the terminal (the server) — useful when they clash with vi or tmux. Choosing a key used by another command clears that command's key.

![Settings > Shortcuts](shortcuts.png)

| Default key | Action |
|---|---|
| Ctrl+K | Quick connect |
| Ctrl+Shift+T | New tab |
| Ctrl+Alt+T | New local terminal |
| Ctrl+Shift+W | Close tab |
| Ctrl+Shift+D / E | Same server right / below |
| Ctrl+Shift+F | Find in terminal |
| Ctrl+Shift+S | SFTP window |
| Ctrl+Shift+B | Broadcast input |
| Ctrl+Tab | Next tab |
| F11 | Maximize/restore window |

## Settings {#settings}

Settings (the gear) are saved as soon as you change them.

![Settings > Appearance](settings.png)

| Section | Contents |
|---|---|
| Appearance | Language, theme, terminal colors, font, cursor, bell, remember window size, maximum split panes |
| Input | Copy on select, paste confirmation, production dangerous-command check |
| Shortcuts | Change keys per command, turn shortcuts off |
| Output rules | Keyword highlighting, output triggers, Windows notifications |
| File transfer | Download folder, temp folder, ZMODEM |
| Connection | Reconnect, keepalive |
| Logs | Session logs |
| git sync | Repository, interval |
| Security | Auto-lock, unlock with Windows sign-in, saved accounts |
| Backup | Export and import |
| About | Version, license, update check |

**"Help for this section"** at the bottom left of Settings opens the help for that section.

## Personal/business use and updates {#license}

- **Personal use** is free. **Business use** needs a license per user. Enter the license key in Settings > About.
- **Updates**: the Microsoft Store version checks the Store; the version installed from the installer checks the public release repository. When a new version exists, a button appears in the status bar.

## FAQ {#faq}

Each question has its own card. Type a word in the search box to see only the topics that contain it.

Q: My connection keeps dropping.
Turn on Settings > Connection "Reconnect when disconnected". Shortening the keepalive interval in Server editor > Advanced can stop firewalls from cutting idle connections. The reason for the disconnect is shown in the terminal.

Q: I want server work to keep running after I close the pane.
Server editor > Advanced > Keep session (tmux).

Q: I want to know when a build in another tab finishes.
Add an output trigger for text such as `BUILD SUCCESS` and turn on notify. When the app is in the background it also arrives as a Windows notification.

Q: I want the `su -` password entered automatically.
In the login script, choose the su password item in [Vault value] of the step that waits for `password`. You don't have to write the password in the send text, and it is not stored as plain text.

Q: SSH does not work behind the company proxy.
Choose HTTP/SOCKS5 in Server editor > Advanced > Proxy, or if your company uses a tunnel program (`ssh -W`, `cloudflared`, ...), set it as the proxy command. `%h` becomes the server address, `%p` the port, and `%r` the user name.

Q: How much went through a port forward?
Click the `⇄ N` button in the pane header to see bytes sent and received per forward (since connecting).

Q: How much did this session send and receive?
The `↑ … ↓ …` on the right of the pane header shows bytes sent (input) and received (output) since the session opened. Hover it for exact values. It keeps counting across reconnects; SFTP and port forwards are counted separately. It is hidden in narrow panes (header 360px or less) — widen the pane or maximize it (Ctrl+Shift+Enter).

Q: The app locks while the screen saver is on.
The app locks along with Windows only when Windows is actually locked. If it still locks, check Settings > Security > Auto-lock (minutes) — it also locks after that long without input.

Q: My PPK key is not read.
Check the passphrase. If it still fails, convert it with PuTTYgen > Conversions > Export OpenSSH key.

Q: Is my information sent anywhere?
No. The server list, passwords, keys and license checks are all handled on this PC only (with git sync on, only encrypted files go to your own repository).
