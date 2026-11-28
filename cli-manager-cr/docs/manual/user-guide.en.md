# DariTerm User Guide

DariTerm is a terminal app for people who work on servers. Use SSH, Telnet, serial devices and your PC's own shells (cmd, PowerShell, WSL) side by side in tabs and split panes. Server passwords and keys are encrypted and stored only on your PC.

> This guide covers version 1.2.0. 한국어: [user-guide.ko.md](./user-guide.ko.md)

---

## 1. Install

1. Run `DariTerm-amd64-installer.exe`.
   - If you see "Windows protected your PC", click **More info → Run anyway** (the installer is not code-signed yet).
2. Allow administrator access, accept the license, and click **Install**.
   - Install folder: `C:\Program Files\DariTerm`
   - A **DariTerm** shortcut is added to the Start menu and the desktop.
3. To uninstall: **Settings > Apps > Installed apps > DariTerm > Uninstall**. Your server list and settings are kept, so reinstalling picks up where you left off.

## 2. First launch

The first launch takes four short steps:

1. **Choose your language** — 10 languages. Your Windows language is preselected.
2. **Choose how you will use it** — *Personal* (free, no time limit) or *Business* (free 30-day evaluation, then a license). You can enter a business license key here if you have one.
3. **Create an app lock password** — at least 8 characters. It encrypts your server passwords and keys and is never sent anywhere.
4. **Save your recovery code** — a 24-character code shown **only once**. Print it or write it down. It is the only way to recover if you forget the password.

> 💡 If you used PuTTY or SecureCRT, DariTerm offers to import your saved sessions (see section 7).

## 3. A quick tour

```
┌───────────┬───────────────────────────────────────────┐
│ Icon row  │ Tab bar  [web-01] [db-01] [+]              │
├───────────┼───────────────────────────────────────────┤
│ Server    │                                           │
│ list      │        Terminal panes (can be split)       │
│ (sidebar) │                                           │
│ Snippets  │                                           │
├───────────┴───────────────────────────────────────────┤
│ Status bar: connection · usage · Unlocked              │
└───────────────────────────────────────────────────────┘
```

- **Top-left icons**: show/hide the server list, snippets, broadcast input to all panes, settings.
- **Top-right icon**: SFTP (file) panel.
- **Pane header**: server name and target, plus a `⇄ N` button when port forwarding is set. The `↑1.2 K ↓34 M` before the buttons on the right is the session's **traffic** — ↑ sent (input), ↓ received (output), since the session opened, updated every second. Hover it for exact values; it is hidden when the pane is narrow.
- Click **"Unlocked"** at the bottom right to lock the app immediately.

## 4. Add a server and connect

### Add a server
1. Click **Add server** in the sidebar.
2. **General** tab: protocol (SSH/Telnet/Serial/Local), display name, address, port, user name.
   - Typing `user@10.0.0.11:22` in the address field fills in user, host and port for you.
   - Set **Environment** to *Production* to get a red border and an extra confirmation before dangerous commands.
   - A server with no color or environment of its own **inherits them from its folder**. If a folder is *Production*, the servers in it are protected as Production too (an environment set on the server itself wins).
3. **Sign-in** tab: password / key file / SSH agent / ask every time.
4. Click **Save and connect**.

### Ways to connect
- **Double-click** a server in the sidebar.
- Press **Ctrl+K** (Quick connect), type part of a name or IP, and press Enter.
  - Type `user@host` to connect to a server you haven't saved.
  - Type `>` for commands (new local terminal, open settings, switch theme, lock app, …).
- **Ctrl+Alt+T** opens a terminal on your PC.

### First connection to a server
You'll see "First connection to this server" with a key fingerprint. If it matches what your server administrator gave you, click **Trust and connect**. If the key changes later you'll get a warning — don't connect unless you know why.

### Save the password
When asked for a password, tick **"Save encrypted and sign in automatically next time"**.

## 5. Using the terminal

| To… | Do this |
|---|---|
| Copy | Select with the mouse — it's copied right away (or Ctrl+Shift+C) |
| Paste | Ctrl+V or right-click |
| **Send a screenshot to the server** | Copy a screenshot, then press **Ctrl+V** in the terminal. DariTerm uploads it and types its path — handy for AI CLIs such as Claude Code |
| Insert a file path | Drag a file from Explorer onto the terminal |
| Find text | Ctrl+Shift+F |
| Change font size | Ctrl+= bigger, Ctrl+- smaller, Ctrl+0 reset |
| New line without running | Shift+Enter |

> ⚠️ **Production protection**: on a *Production* server, commands like `rm -rf`, `reboot` or `DROP TABLE` are not run right away — DariTerm asks "Run it anyway?" first.

## 6. Tabs and split panes

| To… | Do this |
|---|---|
| Open the same server beside it | Ctrl+Shift+D (right), Ctrl+Shift+E (below) |
| Move between panes | Alt+Arrow keys |
| Maximize one pane | Ctrl+Shift+Enter |
| Switch tabs | Ctrl+Tab, Ctrl+1–9 |
| Type the same command into several servers | Ctrl+Shift+B (broadcast input on/off) |
| Close | Ctrl+Shift+W |
| Two side-by-side views | Right-click a tab → "Move to right side", or drag a tab to the right edge and drop it when "Drop here to split left and right" appears |

The tab you are looking at has a color bar on top (its server color, or blue). Other tabs get a light tint of their server color.

## 7. Import from other apps

Click **Import** in the sidebar. DariTerm looks for:

- PuTTY saved sessions
- SecureCRT session folders (including folder structure and port forwarding) — use **Choose SecureCRT folder** for another location
- Xshell sessions (an `.xts` export file or an `.xsh` session file, including folders and port forwarding)
- `~/.ssh/config`
- AWS EC2 — **Load AWS EC2** (requires the aws CLI)

Tick what you want and click **Import N**. Passwords are not imported; enter each once on first connection and save it.

## 8. Transfer files (SFTP)

1. Connect over SSH, then press **Ctrl+Shift+S** (or the top-right icon).
2. The server's files appear on the right:
   - **Upload**: drag files from Explorer onto the list
   - **Download / edit**: right-click a file → Download / "Edit with an app on this PC" (saving uploads it automatically)
   - **Preview**: double-click a file
   - **Sort**: click Name, Size, or Modified above the list; click again to reverse (folders stay on top)
   - **Move terminal to this folder (cd)**
3. **Server-to-server copy**: drag a file from one server's tab to another server's tab. Progress appears in the copy list at the bottom of the window.
4. **If a transfer drops or is cancelled**: an existing file with the same name on the receiving side keeps its old content (DariTerm writes to a temporary `name.dtpart-xxxxxxxx` and renames it only when done). If a network drop leaves a `.dtpart-` file behind, delete it by hand.

## 9. Snippets

Save frequently used commands and run them with one click.

1. In the snippets area below the sidebar, click **New command**.
2. Enter a name and command (multiple lines allowed). Turn on "Run immediately after typing" to run it right away.
3. Click a snippet to type it into the current pane. In a tab with broadcast input on, it goes to every pane.

## 10. Automate repetitive work

| What you want | How |
|---|---|
| Fill in values each time | Write `{{name}}` or `{{name:default}}` in a snippet. DariTerm asks for the values and shows a preview of the filled command |
| Type several steps, waiting for output | Snippets > **Macros**: each step is "when this text appears → send this". You can also **record** what you type in a pane (a recording badge appears in the status bar). In a broadcast tab, macros run in every pane |
| Do it every time you connect | Server editor > **Automation** > Login script: a step like `password:` → `{{password}}` answers `sudo su -` or a device `enable` prompt. For other passwords pick a **vault value** instead of plain text |
| Get notified when output appears | Settings > **Output rules** > Output triggers: on `ERROR`, show a notification, play a sound or reply. When the app is in the background it also appears in the Windows notification center. For one server only, use Server editor > Automation |
| Color words in the output | Settings > Output rules > Keyword highlighting (display only) |

## 11. Port forwarding, jump hosts and proxies

- **Port forwarding**: Server editor > Port forwarding tab. L (a port on your PC → an address reachable from the server), R (a server port → an address on your PC side), D (a port on your PC as a SOCKS proxy). Each forward is reported in one line when you connect, and **`⇄ N`** in the pane header shows status, open connections and traffic.
- **Jump hosts**: Server editor > Advanced > Jump host (ProxyJump). `ProxyJump a,b` in ssh config becomes a jump chain on import.
- **Proxies**: Server editor > Advanced > Connect through a proxy — HTTP CONNECT, SOCKS5, or a **proxy command** (the input and output of an external command such as `ssh -W %h:%p bastion`, `ncat` or `cloudflared`).
- **Agent**: the "SSH agent" sign-in uses both the Windows OpenSSH agent and Pageant. To use this PC's keys inside the server, turn on Advanced > Agent forwarding.

## 12. Transfer files in the terminal (ZMODEM)

In SSH and serial panes, run `sz file` on the server to download and `rz` to upload. The download folder and the on/off switch are in Settings > File transfer. **Ctrl+C** while receiving cancels and returns to the prompt. Transfers cannot work inside tmux or screen, so run them outside.

## 13. Create an SSH key and add it to a server

How to switch a server from password to key sign-in.

1. Settings > Security > Saved accounts > **[Create SSH key]**: a name, the type (Ed25519 recommended, RSA 4096 for older servers) and an optional passphrase. The key is stored encrypted in the vault.
2. In a pane signed in with a password, right-click > **Add public key to this server…**, pick the key and click [Add]. One line is added to `~/.ssh/authorized_keys` on the server (nothing changes if it is already there).
3. Leave "Sign in to this server with this key from next time" on to switch the server settings to key sign-in as well.

[Public key] on a key account shows the public key line and fingerprint to copy (for GitHub and so on).

## 14. Serial devices: hex view, hex send and XMODEM

In the right-click menu of a serial pane.

| What you want | How |
|---|---|
| See received bytes as hex | **Show as hex** — 16 bytes per line like `01 41 FF  |.A.|`. Click again for text |
| Send exact bytes | **Send hex…** — type `01 03 00 00 00 0A C5 CD` and click [Send]. The window stays open for repeated sends |
| Upload firmware | **Send with XMODEM / XMODEM-1K** — pick a file, then start receiving on the device (`loadx` in a boot loader and so on) |
| Get a file from the device | **Receive with XMODEM** — pick a name, then start sending on the device |
| Stop a transfer | Ctrl+C or "Cancel XMODEM transfer" |

## 15. Help window

Click the **question mark** at the top left (or "Help for this section" in Settings) to search feature descriptions. Sort the list by feature order or A–Z, read the FAQ cards and screenshots (click to enlarge), and drag to resize the window.

## 16. Staying secure

- **Auto-lock**: after 15 minutes without input, or when Windows locks, DariTerm locks itself. Connections stay open. Change the time in Settings > Security.
- **Unlock with Windows sign-in**: turn it on in Settings > Security.
- **Forgot the password?** On the lock screen, choose "Reset with recovery code".
- **Manage saved accounts**: Settings > Security > Saved accounts.

## 17. Using several PCs

- **Backup**: Settings > Backup → **Export backup**. On another PC, **Import backup** and unlock with the original PC's app lock password.
- **git sync**: enter a **private** git repository in Settings > git sync. DariTerm uploads one file encrypted with your app lock password to keep server lists in sync.

## 18. Settings at a glance

Settings (gear icon) are saved as soon as you change them.

- **Appearance**: language, theme (dark/light), terminal colors, font, cursor, maximum vertical panes
- **Input**: copy/paste behavior, confirm multi-line paste, production command warning
- **Shortcuts**: change or clear the key for each action, turn all shortcuts on or off
- **Output rules**: output triggers, keyword highlighting
- **File transfer**: server upload folder for images, auto-delete old uploads, ZMODEM download folder
- **Connection**: reconnect when disconnected, restore last layout (reopens your tabs, splits and servers when the app starts)
- **Log**: log terminal output, timestamp each line, keep ANSI codes, retention
- **git sync / Security (including Create SSH key) / Backup / About**

## 19. Personal and business use

- **Personal use** (learning, personal projects, hobbies) is free with no time limit.
- **Business use** (work for a company, institution or organization, or commercial work) can be evaluated free for 30 days; after that each user needs a business license.
- Paste your license key in **Settings > About > Usage** and click **Apply**. The company name and number of licensed users are shown.
- After the evaluation ends the program keeps working; a notice appears once at startup.
- Licensing inquiries: **soli97403@gmail.com**

## 20. FAQ

**Q. My connection keeps dropping.**
With Settings > Connection "Reconnect when disconnected" on, DariTerm retries at 2, 4, 8, 16 and 30 seconds. After it stops, press Enter in the terminal to reconnect.

**Q. I want my work on the server to keep running after I close the window.**
Turn on **"Keep session (tmux)"** in the server editor > Advanced. The server needs tmux. Reopening brings you back to the same screen.

**Q. I need to go through a jump server.**
Server editor > Advanced > "Jump server (ProxyJump)": pick a saved server or enter one manually.

**Q. Port forwarding?**
Server editor > Port forwarding tab: add L, R or D (SOCKS) (§11).

**Q. How much did this session send and receive?**
The `↑ … ↓ …` on the right of the pane header shows bytes sent (input) and received (output) since the session opened, updated every second; hover it for exact values. It keeps counting across reconnects; SFTP and port forwards are counted separately. If the pane is too narrow to show it, widen the pane or maximize it with Ctrl+Shift+Enter.

**Q. Can I use PuTTY keys (.ppk)?**
Yes. PPK formats 2 and 3 (including passphrase-protected keys) are read without conversion. To make a new key, use Create SSH key in §13.

**Q. Is any of my data sent anywhere?**
No. Server lists, passwords, keys and license checks are all handled on your PC.

---

## Keyboard shortcuts

These are the **default keys**. In Settings > Shortcuts you can change or clear the key of each action, and turning "Use shortcuts" off disables all app shortcuts so those keys go to the terminal (server) — useful when vi or tmux use the same keys. If you pick keys another action was using, that action is left without a key. The paste and copy keys cannot be changed and keep working when shortcuts are off.

| Keys | Action |
|---|---|
| Ctrl+K | Quick connect |
| Ctrl+Alt+T | New local terminal |
| Ctrl+Shift+W | Close pane |
| Ctrl+Shift+D / E | Same server to the right / below |
| Alt+Arrow keys | Move between panes |
| Ctrl+Shift+Enter | Maximize pane |
| Ctrl+Tab / Ctrl+1–9 | Switch tabs |
| Ctrl+V | Paste (auto-upload images and files) |
| Ctrl+Alt+V | Clipboard image → server |
| Ctrl+Shift+C | Copy |
| Ctrl+Shift+F | Find in terminal |
| Ctrl+Shift+S | SFTP panel |
| Ctrl+Shift+B | Broadcast input |
| Ctrl+Shift+L | Show/hide server list |
| Ctrl+= / - / 0 | Font size |
| F11 | Maximize/restore window |

© 2026 YangHun Lee (kai). All rights reserved.
