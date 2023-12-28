To update Sindri from __PROTOCOL__://__HOST__, run:

update-sindri

To uninstall Sindri, run:

uninstall-sindri

If you're trying to use a different Sindri than the one that was previously used at __HOST__,
it is recommended to uninstall Sindri using the above command and then reinstall Sindri from
scratch. To do this, run:

uninstall-sindri
curl -fSs http://new-sindri-address/mods.gz | tar -xzf -
