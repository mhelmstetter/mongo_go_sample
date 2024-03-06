Elastic Beanstalk -- to create the application archive for upload
--------------------
cd mongo_go_sample
zip ../gosample.zip -r *


docker build -t go_sample .
docker run -e MONGODB_CONNECTION_STRING="XXXXX" -p 8080:8080 go_sample


docker tag go_sample mhelmstetter/go_sample
docker push mhelmstetter/go_sample

issue with docker run (solution, upgrade docker on mac):
==============================================================
runtime/cgo: pthread_create failed: Operation not permitted
SIGABRT: abort
PC=0x7f9564f1de2c m=0 sigcode=18446744073709551610
