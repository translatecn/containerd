./bin/crictl create f84dd361f8dc51518ed291fbadd6db537b0496536c1d2d6c05ff943ce8c9a54f ./examples/container-config.json ./examples/sandbox-config.json
./bin/crictl runp --runtime=runc ./examples/sandbox-config.json
./bin/crictl inspectp 53675eb8893ee865ccdf2b19caee7f9883a2583b8622fea0f79effb04e73665e
./bin/crictl runp ./examples/container-config.json
./bin/crictl start 3e025dd50a72d956c4f14881fbb5b1080c9275674e95fb67f965f6478a957d60
./bin/crictl run ./examples/container-config.json ./examples/sandbox-config.json


