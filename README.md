# VPVC Executable Wrapper
**Executable wrapper for proximity voice chat for Valorant**

This project creates a standalone executable for [VPVC](https://github.com/thatmarcel/vpvc).

Up until [recently](https://github.com/microsoft/WindowsAppSDK/issues/2597#issuecomment-1930905421) it was not possible to create a single standalone executable for apps made with the Windows App SDK. As a workaround, all necessary files are packaged into a program that extracts and executes these on launch.

The output files from a build of VPVC need to be zipped (without compression), compressed via [S2](https://github.com/klauspost/compress/tree/master/s2), and then saved as `resources/app-archive.zip.s2`. This way the decompression is much faster compared to a normal compressed zip file.