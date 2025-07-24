# Version Checking Configuration Guide

## Overview

The Game Launcher now includes a sophisticated version checking system that allows you to configure exactly how to detect version updates from websites. This replaces the basic pattern matching with a more intelligent and configurable approach.

## How It Works

### 1. **First Run Behavior**
- When you first add a game with a source URL, the system will automatically scan the webpage
- It will try to find any version information using common patterns
- The first version found becomes your "current version" for future comparisons

### 2. **Subsequent Checks**
- The system compares the found version with your stored "current version"
- If they differ, an update is detected
- You can configure exactly where and how to find version information

## Configuration Fields

### **Version Selector (CSS)**
This is a CSS selector that tells the system where to look for version information on the webpage.

**Examples:**
- `.version` - Looks for elements with class "version"
- `#version` - Looks for element with ID "version"
- `h1` - Looks in all H1 headers
- `.download-version` - Looks for elements with class "download-version"
- `[data-version]` - Looks for elements with data-version attribute

### **Version Pattern (Regex)**
This is a regular expression pattern that extracts the version number from the text found by the selector.

**Examples:**
- `v(\d+\.\d+\.\d+)` - Extracts version from "v1.2.3" format
- `(\d+\.\d+\.\d+)` - Extracts version from "1.2.3" format
- `Version (\d+\.\d+)` - Extracts version from "Version 1.2" format
- `(\d+\.\d+\.\d+\.\d+)` - Extracts version from "1.2.3.4" format

### **Current Version**
This is the version you currently have installed. The system compares found versions against this.

## Step-by-Step Configuration

### **Step 1: Find the Version on the Website**
1. Go to the game's download/source page
2. Right-click on the version information and "Inspect Element"
3. Note the HTML structure around the version

### **Step 2: Determine the CSS Selector**
Look at the HTML and find a unique selector for the version element:

```html
<!-- Example 1: Class-based -->
<div class="version">v1.2.3</div>
<!-- Use selector: .version -->

<!-- Example 2: ID-based -->
<span id="current-version">1.2.3</span>
<!-- Use selector: #current-version -->

<!-- Example 3: Data attribute -->
<div data-version="1.2.3">Download</div>
<!-- Use selector: [data-version] -->

<!-- Example 4: Nested structure -->
<div class="download-info">
  <h2>Version 1.2.3</h2>
</div>
<!-- Use selector: .download-info h2 -->
```

### **Step 3: Create the Regex Pattern**
Based on the text format, create a regex pattern:

```text
Text: "v1.2.3" → Pattern: v(\d+\.\d+\.\d+)
Text: "1.2.3" → Pattern: (\d+\.\d+\.\d+)
Text: "Version 1.2.3" → Pattern: Version (\d+\.\d+\.\d+)
Text: "Download v1.2.3" → Pattern: v(\d+\.\d+\.\d+)
```

### **Step 4: Configure in Game Launcher**
1. Edit the game in the launcher
2. Fill in the configuration fields:
   - **Version Selector**: Your CSS selector
   - **Version Pattern**: Your regex pattern
   - **Current Version**: Your current version (optional, will be auto-detected)

## Real-World Examples

### **Example 1: GitHub Release**
**Website**: https://github.com/user/game/releases
**HTML**: `<h1>Release v1.2.3</h1>`
**Configuration**:
- Selector: `h1`
- Pattern: `v(\d+\.\d+\.\d+)`
- Current Version: `1.2.3`

### **Example 2: Game Download Page**
**Website**: https://game.com/download
**HTML**: `<div class="version-info">Current Version: 1.2.3</div>`
**Configuration**:
- Selector: `.version-info`
- Pattern: `Version: (\d+\.\d+\.\d+)`
- Current Version: `1.2.3`

### **Example 3: Steam-like Page**
**Website**: https://game.com
**HTML**: `<span id="version">v1.2.3</span>`
**Configuration**:
- Selector: `#version`
- Pattern: `v(\d+\.\d+\.\d+)`
- Current Version: `1.2.3`

## Testing Your Configuration

### **Method 1: Manual Check**
1. Configure your game with the selector and pattern
2. Click "Check for Updates" in the launcher
3. Look at the description in the update notification
4. It will show "Current: X, Found: Y" to help you verify

### **Method 2: Console Version**
1. Run the console version: `gamelauncher_console.exe`
2. Select "Check for Updates"
3. It will show detailed information about what was found

## Troubleshooting

### **No Version Found**
- Check if the CSS selector is correct
- Verify the element exists on the page
- Try a broader selector (e.g., `h1` instead of `.specific-class`)

### **Wrong Version Extracted**
- Check your regex pattern
- Test the pattern on the actual text
- Use regex testing tools to verify

### **False Update Alerts**
- Make sure your "Current Version" is set correctly
- Check if the website shows different versions in different locations
- Verify the selector isn't picking up multiple elements

## Advanced Tips

### **Multiple Version Elements**
If there are multiple version elements on the page, the system will use the first one found. You can make your selector more specific:

```css
/* Instead of just .version */
.download-section .version
```

### **Dynamic Content**
If the version is loaded dynamically (JavaScript), the scraper might not see it. In this case:
1. Wait for the page to fully load
2. Look for the version in the page source
3. Use a selector that targets the loaded content

### **Complex Patterns**
For complex version formats, you can use more sophisticated regex:

```regex
# For "Version 1.2.3 (Build 456)"
Version (\d+\.\d+\.\d+)

# For "v1.2.3-beta"
v(\d+\.\d+\.\d+)

# For "Release 1.2.3.4"
Release (\d+\.\d+\.\d+\.\d+)
```

## Default Behavior

If you don't configure any selector or pattern:
1. The system will automatically scan for common version patterns
2. It will look in elements with "version" in class/id names
3. It will check headers (h1, h2, h3)
4. The first version found becomes your current version

This makes the system work out-of-the-box for many websites while allowing you to fine-tune it for specific cases.

## Specialized Site Support

### **F95zone Integration**
The launcher includes specialized support for [F95zone](https://f95zone.to/) game threads. When you add a game with an F95zone URL, the system will automatically:

1. **Detect F95zone URLs**: Any URL containing `f95zone.to` triggers specialized parsing
2. **Extract Version Information**: Automatically finds version information in the format "Version: X.X.X.X with RTP"
3. **Handle Multiple Formats**: Supports various version formats like:
   - `0.514.0.3 with RTP`
   - `v1.2.3`
   - `Version 1.2.3`
   - `1.2.3.4`

**Example F95zone Configuration:**
```
Game: Warlock and Boobs
Source URL: https://f95zone.to/threads/warlock-and-boobs-v0-514-0-3-boobsgames.18692/
Current Version: 0.514.0.3
```

**No manual configuration needed** - the system automatically detects and extracts version information from F95zone threads!

### **GitHub Integration**
GitHub repositories are also automatically detected and handled using the GitHub API for release information. 