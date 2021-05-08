set -eux
rm -rf ~/.smartbchd
# shellcheck disable=SC2216
rm out.txt | echo "no file"
go build -o build/smartbchd ./cmd/smartbchd
./build/smartbchd init freedomMan --chain-id 0x2711
cp ~/config.toml ~/.smartbchd/config
cp ~/genesis.json ~/.smartbchd/config
./build/smartbchd start
