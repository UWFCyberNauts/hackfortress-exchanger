## Hackfortress Exchanger
This component of the UWF Cyber Security Clubs hackfortress competition implementation allows 
for exchanging data from the [hackfortress-sourcemod-extension](https://github.com/UWFCybernauts/hackfortress-sourcemod-extension)
with external services utilizing gRPC. This component hosts a UNIX Domain Socket that the
[hackfortress-sourcemod-extension](https://github.com/UWFCybernauts/hackfortress-sourcemod-extension)
connects to. Any communication from the SourceMod side goes through this UNIX Domain Socket
and get's recieved by this component. This component can then perform actions based upon the data
arriving through the UNIX Domain Socket. 

### Reasons for this component
We have implemented these components this way in order to avoid some issues that were occuring due 
to libstdc++, parrellization in SourceMod/TF2, and reconfiguring of entire libraries to match
compatible compile targets enforced by SourceMod/TF2. This method of implementation also allows
any exchanger to be written in any higher level language that has a method of interfacing with 
UNIX Domain Sockets. 

### Build instructions
This code requires the linux platform
1) Install [gRPC](https://grpc.io)
2) Issue the command `make` in the root of the repo and watch the magic happen

the build binary is output to ./build/out/hackfortress-exchanger

### Authors
This component of UWF Cyber Security Club's Hackfortress Implementation is created and maintained 
by:
* Michael Mitchell (@AWildBeard)
