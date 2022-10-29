# ImageSort

It's a rapid image sorter and semi-competent viewer. The original version was written in Python over a few weekends. Due to issues with CPU usage, I switched to Golang and SDL 2.

## Building

In addition to what is needed from `go mod download`, the following dependencies are required to build:

- [SDL](https://github.com/libsdl-org/SDL) v2.0 or later
- libavcodec, libavformat, libswscale, libavutil v4.2.7 or later (this corresponds to FFmpeg version/package version, not library version)
  - For Linux, install the dev packages from your package manager of choice
  - For Windows, download the windows-shared build from [BtbN](https://github.com/BtbN/FFmpeg-Builds/releases) and install the libraries and headers.
  - If you do not want to use libav, switch to branch `ffmpeg`.

## Usage

Upon opening the application, it displays a list of subfolders of the folder it's in. If you select a subfolder, it will open it in the image browser. In the image browser, you can view and zoom images to ensure that they are in the correct folder. If they are not in the correct folder, you can send them to the Sort folder. If you do not like the image, you can send it to the Trash. Images in Trash cannot be individually deleted, you can only delete the entire folder.

In the Sort folder, there is a folder bar at the top of the UI listing every folder except for Sort and Trash. Pressing Q will scroll this bar forward. Pressing a number key will move the image to the corresponding folder on the top bar.

In the deduplicator, you view images in sets of two. Press the Q key to switch between the two images. Pressing Z, X, C, V, or H will perform the operation only on the currently active image.

## Controls

### Folder Menu

- Up/Down arrows - Change selection
- Enter - Pick folder/submenu
- D - Delete an empty folder
- R - Open the deduplicator on the highlighed folder
- U - Open the deduplicator on all folders except Trash
- ESC - Close the program

### Image Browser

- ESC - Return to folder menu
- Left/Right arrow - Change image
- Up/Down arrow - Zoom
- WASD - Move zoomed image
- Z - Image info
- X - Send image to Sort folder
- C - Send image to Trash folder
- V - Open image in external application
- H - Highlight image in folder
- G - Goto image
- Home/End - Go to first/last image

### Trash Folder

Similar to the image browser, but...

- C - Nothing
- L - Empties the trash

### Sort Folder

Similar to the image browser, but...

- X - Nothing
- Q - Scroll folder bar forward. Will loop at the end.
- 1-9, -, = - Move to corresponding folder on folder bar
- I - Hide/show folder bar

### Create Folder

- Enter - Submit folder name
- ESC - Cancel

### Deduplicator

Similar to image browser, but...

- Q - Switch images

### Options Menu

- Up/Down Arrow - Change selection
- Left/Right Arrow - Modify value
- ESC - Back to folder menu

## Options explanation

- Fade Speed: How fast the transition between screen is. Higher is faster.
- Dupe sensitivity: How many bits of the hash can be different before two images are declared dissimilar.
- Sample Size: Controls the size of the image hashes used by the DeDuplicator. Changing this will require all images to be rehashed.

## Known Bugs

- After moving to a new pair in the deduplicator, the background will change to the default color even if this should not happen.
- When deleting an image or fading to another screen in the Sort folder, the folder bar is not visible.
