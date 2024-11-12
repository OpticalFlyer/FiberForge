# FiberForge
An experimental GIS/CAD project written in Go with Ebitengine.

Execute and complete commands with space or return/enter.  Space or return without a command executes last command.  For example, PL`<space>` to begin a polyline, `<space>` to complete the polyline.  `<space>` again by itself to start a new polyline.

### Command List

PL - Draw poly line  
PO - Draw point  
POL - Draw polygon  

OSM - OpenStreetMap base map  
GOOGLEAERIAL - Google aerial base map  
GOOGLEHYBRID - Google hybrid base map  
BINGAERIAL - Bing aerial base map  
BINGHYBRID- Bing hybrid base map  

MAPIMPORT - Load KML or KMZ from file path on clipboard

Drag and drop support loading KML/KMZ.  Just drag the file to the window to load.
