# DariTerm behavior scenarios

> English version of [scenarios.md](./scenarios.md) (as of 2026-11-14, app 0.1.0). Format: "situation → what you do → what the app does".

## Contents

1. Install and first launch
2. App lock
3. Adding and organizing servers
4. Connecting
5. Authentication and security prompts
6. Using the terminal
7. Tabs and split panes
8. Working with files
9. Snippets
10. Importing from other apps
11. Sync and backup across PCs
12. Disconnects and keeping sessions
13. Settings
14. Language
15. Usage and license
16. Quitting and restarting

---

## 1. Install and first launch

| # | Situation | What you do | What the app does |
|---|---|---|---|
| 1-1 | First install | Run `DariTerm-amd64-installer.exe` | Asks for administrator rights, then shows the installer in your Windows language (10 languages, English otherwise) → license agreement → install folder (default `C:\Program Files\DariTerm`) → install. Installs WebView2 if missing. Adds Start menu and desktop shortcuts and an Installed apps entry (publisher YangHun Lee). |
| 1-2 | App running during install/uninstall | Run the installer | "DariTerm is running" → close it and click [Retry], or [Cancel]. A silent install (/S) cancels right away. |
| 1-3 | First launch | Start the app | ① **Language** (Windows language preselected, preview on click) → ② **Usage** (Personal/Business, optional key) → ③ **Setup** (app lock password, 8+ characters, strength meter) → ④ **Recovery code** (24 characters, shown only here; tick "I saved the recovery code somewhere safe" → [Get started]) → start screen. |
| 1-4 | After first launch | — | If there are no servers yet and PuTTY saved sessions or `~/.ssh/config` exist, the **Import servers window opens once automatically** (not asked again). |
| 1-5 | No tabs open | — | Start screen: [Quick connect] [Local terminal] [Add server] [Import from PuTTY] and a table of 12 shortcuts. |
| 1-6 | Uninstall | Installed apps → DariTerm | Removes the program folder and shortcuts. **Server list, settings and encrypted accounts (`%APPDATA%\DariTerm\data`) are kept.** |

## 2. App lock (protects server passwords and keys)

| # | Situation | What you do | What the app does |
|---|---|---|---|
| 2-1 | App start | — | Lock screen "DariTerm is locked". Enter the app lock password → [Unlock]. |
| 2-2 | Unlock with Windows | Settings > Security "Unlock with Windows sign-in" on | The lock screen shows [Unlock with Windows account] (DPAPI). |
| 2-3 | Stepping away | — | Locks automatically after 15 minutes without input (setting, 0 = off) or when Windows locks/sleeps. **Connected sessions stay open** (the lock screen covers them). |
| 2-4 | Lock now | Click "Unlocked" in the status bar, or Quick connect `> Lock app` | Locks immediately. |
| 2-5 | Wrong password | — | "The password is incorrect". |
| 2-6 | Forgot password | [Forgot the password? Reset with recovery code] | 24-character recovery code + new password → saved accounts are kept. |
| 2-7 | Change password | Settings > Security > Change app lock password | Current, new, confirm → "Changed". |
| 2-8 | Manage saved accounts | Settings > Security > Saved accounts | Type (password/key) and servers using each account, [Rename], delete (those servers ask again on next connection). |

## 3. Adding and organizing servers (sidebar)

| # | Situation | What you do | What the app does |
|---|---|---|---|
| 3-1 | Add a server | Sidebar [Add server] / right-click a folder "Add server here" | Server editor: **General** (protocol SSH·Telnet·Serial·Local, display name, address·port·user, folder, tags, environment, color, notes) · **Sign-in** (SSH: password / key file / SSH agent / ask every time; save for this server only or share a saved account) · **Advanced** (jump server, keep session with tmux, command after connecting, encoding, terminal type, keepalive interval) · **Port forwarding** (L/R). [Save] / [Save and connect]. |
| 3-2 | `user@host:22` in the address field | Type it | Splits it into user, host and port. |
| 3-3 | Saving a password while locked | [Save] | "The app is locked, so the password cannot be saved". |
| 3-4 | PPK v3 key | Choose key file | "Convert PPK v3 keys with PuTTYgen → Conversions → Export OpenSSH key". OpenSSH·PEM·PPK v2 work (key contents are stored encrypted). |
| 3-5 | Folders | [Add folder], right-click a folder (New subfolder · Edit folder · Delete) | Servers inherit the folder's environment color. Delete: empty folders go immediately; with servers you choose [Delete servers too] / [No (delete folder only — servers move up)] / [Cancel]. |
| 3-6 | Environment | Environment = Production/Staging/Load test/Development/Network device | Shown as a color bar and pill. **Production** gets a red border and a dangerous-command check (6-6). |
| 3-7 | Search | Sidebar "Search servers" | Filters by name, address and tags. Otherwise "No servers match …". |
| 3-8 | Favorites and recent | Right-click a server "Add to favorites" | "Favorites" and "Recent" sections at the top of the list. |
| 3-9 | Move | Drag servers/folders onto another folder | Reorders/moves and saves right away. |
| 3-10 | Select several | Ctrl+click / Shift+click | "N servers selected" bar. Drag them into a folder, or right-click "Delete (N)". |
| 3-11 | Server context menu | — | Open in new tab · Open in right view · Open in pane below · Choose where to open… · Edit (F2) · Duplicate · Favorites · Copy address · Delete. |
| 3-12 | Open a whole folder | Right-click a folder "Open all folder servers tiled" | Tiles them in one tab. Asks first if there are more than 16. |
| 3-13 | Hide the sidebar | Ctrl+Shift+L, top-left icon | Show/hide. Drag the edge to resize (saved). |

## 4. Connecting

| # | Situation | What you do | What the app does |
|---|---|---|---|
| 4-1 | Saved server | Double-click / Enter / Quick connect | Connects in a new tab. The terminal shows "● Connecting to name (target)...". |
| 4-2 | Quick connect | Ctrl+K | Type a name, IP or `user@host:port`. Each word is matched against name > address > folder·tags > user > scattered letters, up to 50 results. Enter = new tab, Ctrl+Enter = split right, Shift+Enter = split down. |
| 4-3 | Unsaved server | Type `user@1.2.3.4` in Quick connect | "Connect now: … (Connect once without saving)". |
| 4-4 | Commands | Type `>` in Quick connect | New local terminal · Add server · New folder · Import · Open settings · Switch theme · Tile all tabs · Lock app. |
| 4-5 | Shell on this PC | Ctrl+Alt+T, start screen [Local terminal] | cmd, PowerShell, WSL, etc. (shell and start folder can be set on a local server). If the start folder is missing it opens in the home folder and says so. |
| 4-6 | Serial device | Protocol Serial | Port, speed, data bits, parity, stop bits, flow control, Enter key value, local echo. |
| 4-7 | Telnet | Protocol Telnet | Port 23. |
| 4-8 | Choose where to open | Right-click a server "Choose where to open…" | Shows the pane layout of open tabs: [Here] on an empty pane, arrows to split a pane in a direction, or [Open in new tab]. |
| 4-9 | Jump server | Advanced > pick a jump server or "Enter manually…" | Connects through it ("Connecting via jump server …"). An overly long chain is an error. |
| 4-10 | Port forwarding | Add L/R rows in the Port forwarding tab | After connecting: "Forwarding A → B". On failure: "Port forwarding L3307 failed: …". |
| 4-11 | Connection failed | — | "Connection failed: " + a plain reason (refused port, timeout, host name not found, network unreachable, authentication failed, …). |

## 5. Authentication and security prompts

| # | Situation | What the app does |
|---|---|---|
| 5-1 | First connection to a server | "First connection to this server" + key fingerprint → [Trust and connect] / [Cancel connection]. Trusted keys are saved. |
| 5-2 | Server key changed | "Warning: the server host key has changed" (possible man-in-the-middle attack) + new and old fingerprints → [Accept the risk and connect with the new key] / [Cancel connection]. |
| 5-3 | Password needed | "Enter password" + [Save encrypted and sign in automatically next time] (not while locked). Asks again if wrong. |
| 5-4 | Extra authentication (OTP, etc.) | An input box with the server's own prompt. |
| 5-5 | Key passphrase | "Enter key passphrase". |
| 5-6 | No user name | "Enter user name". |

## 6. Using the terminal

| # | Situation | What you do | What the app does |
|---|---|---|---|
| 6-1 | Copy | Drag-select / Ctrl+Shift+C | With "Copy on select" on (default), selecting copies right away (PuTTY style). |
| 6-2 | Paste | Ctrl+V / right-click | Pastes text. If Ctrl+V paste is turned off, Ctrl+V goes to the server and Ctrl+Shift+V pastes. If right-click paste is off, right-click opens a menu (Shift+right-click always does). |
| 6-3 | Multi-line paste | Paste 2+ lines | Asks "Paste multiple lines — Pasting N lines into [session]" (can be turned off). |
| 6-4 | Send a screenshot | Copy an image, then Ctrl+V (or Ctrl+Alt+V) | SSH: uploads it as `dt-…` to the server upload folder (default `~/.dariterm/uploads`, created if missing) and **types the path into the terminal** — CLIs such as Claude Code read it as an image. Local: saves it on this PC and types the path. Telnet/serial: "Images can only be sent in a connected SSH session". |
| 6-5 | Insert a file path | Copy files in Explorer then Ctrl+V, or drag onto the terminal | SSH uploads and types the path; otherwise types the original path. Paths with spaces are quoted (setting). |
| 6-6 | Dangerous command on production | On a Production server, type `rm -rf`, `reboot`, `DROP TABLE`, … and press Enter | Enter is held and "Dangerous command on production — Run it anyway?" appears → [Run] / [Cancel]. Pastes are checked too. Patterns are edited in Settings > Input. |
| 6-7 | Find | Ctrl+Shift+F | Find bar with [Previous] [Next] [Close]. |
| 6-8 | Font size | Ctrl+= / Ctrl+- / Ctrl+0 | 8–32; Ctrl+0 resets to 14. Saved in settings. |
| 6-9 | Multi-line input (Claude Code, etc.) | Shift+Enter | New line without running (setting, on by default). |
| 6-10 | Server writes the clipboard | tmux, vim, etc. copy via OSC 52 | Put on this PC's clipboard (setting; read requests are always refused). |
| 6-11 | Log output | Settings > Connection "Log terminal output" on (off by default) | From connections opened afterwards, output is saved to text files. [Open log folder]. |
| 6-12 | Pane context menu | Right-click a pane | Clear screen, Select all, Copy, Paste, Paste image, Find, Same server to the right/below, Open SFTP (file) panel, Reconnect, Close pane. |

## 7. Tabs and split panes

| # | Situation | What you do | What the app does |
|---|---|---|---|
| 7-1 | Same server again | Ctrl+Shift+D (right) / Ctrl+Shift+E (below) | Opens the focused pane's server in a new split. At the maximum vertical panes (default 3, 1–5) it opens in a new tab instead. |
| 7-2 | Empty pane | Pane header [Split down] | Empty pane: drag a server onto it, select it and double-click a server in the sidebar, or [Quick connect…]. |
| 7-3 | Move between panes | Alt+Arrow keys | Focuses the nearest pane. |
| 7-4 | Maximize a pane | Ctrl+Shift+Enter / header [Maximize] | That pane only ↔ restore. |
| 7-5 | Switch tabs | Ctrl+Tab, Ctrl+Shift+Tab, Ctrl+1–8, Ctrl+9 (last), Ctrl+Shift+←/→ (reorder) | |
| 7-6 | Tab menu | Right-click a tab | Rename tab (also double-click) · Move to right/left side · Merge into one view · Broadcast input · Split panes into tabs · Tile all tabs in one view · Close other tabs · Close tab. |
| 7-7 | Two side-by-side views | Drag a tab to the right edge / "Move to right side" | Tabs split into left and right groups (max 2). Drag the divider to resize. |
| 7-8 | Same command to many panes | Ctrl+Shift+B (this tab), top-left chain icon (all visible panes) | "Broadcast on" → input goes to every pane. Only with 2+ panes. |
| 7-9 | Close a tab | Ctrl+Shift+W, the tab's X, middle-click | With several connections: "Close all N connections in this tab?". |
| 7-10 | Reopen an open server | Pick an already-open server in the sidebar | Jump to that pane or open another (Open beside / Open new buttons). |

## 8. Working with files

| # | Situation | What you do | What the app does |
|---|---|---|---|
| 8-1 | Browse server files | Ctrl+Shift+S / top-right icon | SFTP panel: files of the current pane's SSH session (name, size, modified). Parent folder, home, path field, refresh, new folder, upload, [Move terminal to this folder (cd)]. |
| 8-2 | Local terminal files | Same shortcut in a local session | "Files" panel: this PC's start folder. |
| 8-3 | File context menu | — | Open/Preview · Download (local: copy to another folder) · Edit with an app on this PC (auto-upload on save) · Type path into terminal · cd · Copy path · Rename · Change permissions (not on local Windows) · Delete (permanent-delete confirmation). |
| 8-4 | Preview | Double-click a file | Text and images up to 512 KB. Otherwise "Preview is not available for this file". |
| 8-5 | Upload | Drag files from Explorer onto the SFTP list | "Drop here to upload to …" → progress in the transfer list. |
| 8-6 | Server → server copy | Drag from one tab's SFTP list to another server's tab (terminal or list) | Asks before overwriting the same names. Copies through a temp folder on this PC (Settings > File transfer) and deletes the temp files afterwards. The **copy list** at the bottom shows phase, progress, [Cancel], [Clear finished]. A folder cannot be copied into itself. |
| 8-7 | Clean up old uploads | Settings > File transfer | Deletes uploads starting with `dt-` after N days (default 7, 0 = off). |

## 9. Snippets (frequently used commands)

| # | Situation | What you do | What the app does |
|---|---|---|---|
| 9-1 | Add | [New command] in the snippets area below the sidebar | Name, command (multiple lines allowed), "Run immediately after typing". |
| 9-2 | Run | Click a snippet | Types it into the current pane — into every pane in a tab with broadcast on. Without a session: "Select a session first". |
| 9-3 | Search, edit, delete | "Search commands", pencil and trash icons | Delete asks first. |

## 10. Importing from other apps

| # | Situation | What you do | What the app does |
|---|---|---|---|
| 10-1 | Open Import | Sidebar [Import], Quick connect `> Import`, start screen | Finds PuTTY saved sessions (registry), SecureCRT session folders and `~/.ssh/config` and lists them. |
| 10-2 | Choose | Tick items, select all | Name, address, type, source, key file. "Target folder name" (default "PuTTY import"; names starting with prod-/stg-/dev- get that environment). |
| 10-3 | SecureCRT elsewhere | [Choose SecureCRT folder] | Reads a Config folder copied from another PC too, keeping folder structure and port forwarding. |
| 10-4 | AWS | [Load AWS EC2] | All regions via your aws CLI credentials. Private IPs go through a bastion in the same VPC as the jump server; keys from `~/.ssh/<key name>.pem`. |
| 10-5 | Import | [Import N] | "Imported N servers (M keys saved encrypted)". Passwords are not imported (enter once on first connection). |

## 11. Sync and backup across PCs

| # | Situation | What you do | What the app does |
|---|---|---|---|
| 11-1 | Turn on git sync | Settings > git sync: a private repository address | Uploads one file encrypted with the app lock password. git authentication uses this PC's git setup (SSH keys, credential helper). Addresses with passwords or tokens are refused. |
| 11-2 | Automatic sync | — | On unlock, about 20 seconds after a change, and every 10 minutes. [Sync now]. "Last sync: …". |
| 11-3 | Data differs from another PC | — | "The encrypted data in the repository differs from this PC" → [Get repository data] (new PC) / [Overwrite with this PC's data]. |
| 11-4 | Backup | Settings > Backup [Export backup] | Servers, settings, snippets and **encrypted** accounts in one `.dtbackup` file. |
| 11-5 | Restore | [Import backup] | Confirms "Your current server list and credentials will be replaced" → the app locks; unlock with **the app lock password of the PC that made the backup**. |

## 12. Disconnects and keeping sessions

| # | Situation | What the app does |
|---|---|---|
| 12-1 | Connection drops | "Disconnected: …", then reconnects at 2, 4, 8, 16, 30 s, up to 5 times (setting). Serial does not reconnect automatically. |
| 12-2 | Reconnecting gave up | "Press Enter to reconnect." → Enter reconnects immediately. The pane header [Reconnect] does the same. |
| 12-3 | First connection failed | No automatic reconnect; the error is shown. |
| 12-4 | Keep session with tmux | Server Advanced "Keep session (tmux)": programs on the server keep running after you close the pane or app, and reopening reattaches to the same screen. Without tmux it says so and works normally. On local Windows, WSL shells only. |
| 12-5 | Local shell exits | "Shell exited" → Enter starts a new shell. |

## 13. Settings (gear icon, 8 sections)

| Section | Main items |
|---|---|
| Appearance | Language · app theme (Dark/Light) · 6 terminal color themes (DariTerm Dark·Light, PuTTY Classic, Solarized Dark, Monokai, Nord) · font, size, line spacing · remember window size and position · [Reset to default size] · cursor style · scrollback lines · maximum vertical panes · blinking cursor · bell sound |
| Input | Copy on select · paste with right click · paste with Ctrl+V · new line with Shift+Enter · OSC 52 · ask before pasting multiple lines · production dangerous-command warning and pattern list |
| File transfer | Server upload folder · auto-delete days · temp folder for server-to-server copies · quote paths |
| Connection | Reconnect when disconnected · restore last layout · log terminal output · log folder |
| git sync | Repository address · Sync now · automatic sync |
| Security | Auto-lock (minutes) · unlock with Windows sign-in · change password · saved accounts |
| Backup | Export · Import |
| About | Version · developer · license · **Usage (license key)** · terms of use · business license inquiries · referenced project · open source licenses · data folder path (display only) · shortcut table (update checking is currently turned off and hidden) |

Every setting is saved as soon as you change it (no Save button).

## 14. Language

| # | Situation | What the app does |
|---|---|---|
| 14-1 | First launch | Language selection (10). Your Windows language is preselected; English if none matches. |
| 14-2 | Change | Settings > Appearance > Language → the interface and messages change immediately. Open terminals stay. |
| 14-3 | Automatic names | Tab names the app assigned, such as "Local terminal", appear in the current language. |

## 15. Usage and license

| # | Situation | What the app does |
|---|---|---|
| 15-1 | Personal | No time limit. Status bar "Personal". Without a key it is shown as Personal. |
| 15-2 | Business evaluation | 30 days. Status bar "Business trial Nd". |
| 15-3 | Evaluation ended | One notice at startup ([OK] only) → keeps working normally. |
| 15-4 | Enter a license key | Settings > About > Usage field → [Apply]. "Business license: company · number · N licensed users · version range · expiry"; status bar "Business · company (N users)". Otherwise "The license key is not valid". |
| 15-5 | Key expired / version not covered | One notice at startup → keeps working normally. |
| 15-6 | Personal on a company PC | On a domain-joined PC, asks once "This looks like a company PC" → [Switch to Business] / [It is personal use]. Checked only on this PC. |
| 15-7 | Buying | About shows "Business license inquiries: soli97403@gmail.com" [Copy]. |

## 16. Quitting and restarting

| # | Situation | What the app does |
|---|---|---|
| 16-1 | Closing with connections | "There are N connected sessions. Close them all and quit?" → [Quit] / [Cancel]. Mentions panes kept with tmux. |
| 16-2 | Next launch | Restores window size and position (setting), and **restores the last tabs and panes and reconnects to the same servers** (setting "Restore last layout"; passwords, screen contents and unsaved one-off SSH are not restored). |
| 16-3 | Window off-screen | If the monitor layout changed, it opens centered. |
| 16-4 | Maximize | F11: maximize ↔ restore the window (not true full screen). Works even with a dialog open. |

---

### Appendix: Keyboard shortcuts

| Keys | Action |
|---|---|
| Ctrl+K (Ctrl+Shift+T, Ctrl+Shift+N) | Open/close Quick connect |
| Ctrl+Alt+T | New local terminal |
| Ctrl+Shift+W | Close pane |
| Ctrl+Shift+D / E | Same server to the right / below |
| Alt+Arrow keys | Move between panes |
| Ctrl+Shift+Enter | Maximize/restore pane |
| Ctrl+Tab, Ctrl+Shift+Tab, Ctrl+1–9 | Switch tabs |
| Ctrl+Shift+←/→ | Reorder tabs |
| Ctrl+V / Ctrl+Alt+V | Paste (auto-upload images and files) / clipboard image → server |
| Ctrl+Shift+C | Copy |
| Ctrl+Shift+F | Find in terminal |
| Ctrl+Shift+S | SFTP (file) panel |
| Ctrl+Shift+B | Broadcast input |
| Ctrl+Shift+L | Show/hide server list |
| Ctrl+= / Ctrl+- / Ctrl+0 | Font size |
| F11 | Maximize/restore window |
