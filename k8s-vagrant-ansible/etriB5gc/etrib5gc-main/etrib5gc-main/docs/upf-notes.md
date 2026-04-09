## Configure iptables

```bash
sysctl -w net.ipv4.ip_forward=1

iptables -I INPUT -i upfgtp -p udp --dport 2152 -j ACCEPT
iptables -A FORWARD -i upfgtp -o eno1 -j ACCEPT
iptables -A FORWARD -i eno1 -o upfgtp -m state --state ESTABLISHED,RELATED -j ACCEPT
iptables -t nat -A POSTROUTING -s 10.60.0.0/16 -o eno1 -j MASQUERADE
```

## If you are using UFW, make sure routed traffic is allowed

```bash
ufw route allow in on upfgtp out on ens3
ufw route allow in on ens3 out on upfgtp
ufw allow proto udp from 60.60.0.0/16 to any port 53
ufw allow icmp
```

