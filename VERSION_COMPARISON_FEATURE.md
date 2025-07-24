# Version Comparison Feature

## 🆕 New Feature: Real-time Version Comparison

The Game Launcher now includes a real-time version comparison system that automatically fetches and compares versions from source URLs.

## 📊 New UI Layout

The game list now includes two version columns:

1. **Current Version** - Your locally installed version
2. **Fetched Version** - Version fetched from the source URL with status indicators

## 🎨 Status Indicators

The fetched version column shows different status indicators:

### **✓ Green Status (Up to Date)**
- **Format**: `1.2.3 ✓`
- **Meaning**: Your current version matches the fetched version
- **Action**: No action needed

### **⚠ NEW (Update Available)**
- **Format**: `1.2.4 ⚠ NEW`
- **Meaning**: A newer version is available
- **Action**: Consider updating your game

### **⚠ DIFF (Version Mismatch)**
- **Format**: `1.2.1 ⚠ DIFF`
- **Meaning**: Different version found, but not necessarily newer
- **Action**: Check if this is the correct version

### **Other Statuses**
- **`Checking...`** - Currently fetching version information
- **`Error`** - Failed to fetch version (network/parsing error)
- **`Not found`** - No version information found on the page
- **`No source`** - No source URL configured for this game

## 🔄 Automatic Updates

### **Real-time Checking**
- Version checks happen automatically when the application starts
- Each game with a source URL is checked in the background
- Results are displayed as they become available

### **Manual Refresh**
- Use the refresh button (🔄) in the toolbar to check all games again
- Individual games are re-checked when the list is refreshed

## 🎯 How It Works

### **1. Automatic Detection**
- **F95zone URLs**: Automatically detected and parsed using specialized logic
- **GitHub URLs**: Uses GitHub API for release information
- **Other URLs**: Uses generic web scraping with configurable selectors

### **2. Version Comparison**
- Compares fetched version with your current version
- Shows clear status indicators for easy identification
- Updates in real-time as checks complete

### **3. Background Processing**
- Version checks run in background threads
- UI remains responsive during checks
- Notifications show when checks complete

## 📋 Example Display

```
Game Name          | Current Version | Fetched Version
-------------------|-----------------|------------------
Warlock and Boobs  | 0.514.0.3      | 0.514.0.3 ✓
MyPigPrincess      | 0.9.0          | 0.9.1 ⚠ NEW
Another Game       | 1.0.0          | Error
No Source Game     | 1.2.3          | No source
```

## ⚙️ Configuration

### **Version Selectors**
You can configure custom version selectors for better accuracy:

1. **Edit a game** to access version configuration
2. **Set Version Selector**: CSS selector (e.g., `.version`, `#version`)
3. **Set Version Pattern**: Regex pattern (e.g., `v(\d+\.\d+\.\d+)`)
4. **Set Current Version**: Your installed version for comparison

### **F95zone Integration**
F95zone URLs are automatically handled with specialized parsing:
- Detects version format: `0.514.0.3 with RTP`
- No manual configuration required
- Works with all F95zone game threads

## 🚀 Benefits

### **Quick Overview**
- See all game versions at a glance
- Identify which games need updates
- Spot version mismatches immediately

### **Automated Monitoring**
- No manual checking required
- Real-time status updates
- Background processing doesn't block UI

### **Smart Detection**
- Automatic site-specific parsing
- Configurable for custom websites
- Handles various version formats

## 🔧 Technical Details

### **Performance**
- Background thread processing
- Non-blocking UI updates
- Efficient caching of results

### **Error Handling**
- Network timeouts
- Parsing errors
- Missing version information
- Invalid URLs

### **Compatibility**
- Works with all existing games
- Backward compatible with old data
- No migration required

## 🎮 Usage Tips

### **For Regular Users**
- Just launch the application to see version status
- Use the refresh button to check for updates
- Look for ⚠ NEW indicators to find games needing updates

### **For Power Users**
- Configure custom version selectors for better accuracy
- Use the edit dialog to fine-tune version detection
- Monitor the status indicators for version mismatches

### **For F95zone Users**
- Simply add F95zone URLs - no configuration needed
- Automatic version detection and comparison
- Real-time update notifications

This feature makes it easy to keep track of all your games and know exactly which ones need updates! 🎯✨ 