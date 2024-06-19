./bin/crictl create f84dd361f8dc51518ed291fbadd6db537b0496536c1d2d6c05ff943ce8c9a54f ./examples/container-config.json ./examples/sandbox-config.json
./bin/crictl runp --runtime=runc ./examples/sandbox-config.json
./bin/crictl inspectp d3ca6410f31af031333fdaf46997241e381a5c97b02dba3922dd29cdc1d20141
./bin/crictl runp ./examples/container-config.json
./bin/crictl start 3e025dd50a72d956c4f14881fbb5b1080c9275674e95fb67f965f6478a957d60
./bin/crictl run ./examples/container-config.json ./examples/sandbox-config.json


