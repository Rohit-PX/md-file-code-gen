# md-file-code-gen
Library to parse and extract kubectl/pxctl commands from `.md` files then execute and validate them

### How to run
* Clone the repository

* Based on your local OS run the following commands to validate md files

### For MacOS
```
     ./bin/mac/validateDoc -commandType kubectl  -kubeconfig <path-to-kubeconfig> -ipaddr <ip-address-worker>  -mdfile snapdoc.md 
```

### For Linux
```
     ./bin/linux/validateDoc -commandType kubectl  -kubeconfig <path-to-kubeconfig> -ipaddr <ip-address-worker>  -mdfile snapdoc.md 
```