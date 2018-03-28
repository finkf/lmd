from ubuntu
maintainer Florian Fink <finkf@cis.lmu.de>
copy data/*.gz /data/
copy lmd /app/lmd
cmd /app/lmd --host '0.0.0.0:8181' --dir /data
