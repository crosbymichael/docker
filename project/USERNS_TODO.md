This is a todo list for implementing user namespace support in docker.  
The general solution is to add a daemon level flag that allows a user to specify the 
global root uid:gid mapping for user namespaces.  Image file ownership will be remapped 
on push and pull during the pack and unpack of the tar archives.


TODO:

* Update docker with libcontainer API changes.
    - Update native exec driver to use new API.
* Modify `pkg/archive` to support remapping of uid:gid during tar and untar.
* Map uid:gid on tar and untar for image push and pull.
* Add daemon flag for uid:gid specification.
