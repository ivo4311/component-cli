## components-cli component-archive component-references add

Adds a component reference to a component descriptor

### Synopsis


add adds component references to the defined component descriptor.
The component references can be defined in a file or given through stdin.

The component references are expected to be a multidoc yaml of the following form

<pre>

---
name: 'ubuntu'
componentName: 'github.com/gardener/ubuntu'
version: 'v0.0.1'
...
---
name: 'myref'
componentName: 'github.com/gardener/other'
version: 'v0.0.2'
...

</pre>


```
components-cli component-archive component-references add [flags]
```

### Options

```
  -a, --archive string             path to the component archive directory
      --component-name string      name of the component
      --component-version string   version of the component
  -h, --help                       help for add
      --repo-ctx string            repository context url for component to upload. The repository url will be automatically added to the repository contexts.
  -r, --resource string            The path to the resources defined as yaml or json
```

### Options inherited from parent commands

```
      --cli                  logger runs as cli logger. enables cli logging
      --dev                  enable development logging which result in console encoding, enabled stacktrace and enabled caller
      --disable-caller       disable the caller of logs (default true)
      --disable-stacktrace   disable the stacktrace of error logs (default true)
      --disable-timestamp    disable timestamp output (default true)
  -v, --verbosity int        number for the log level verbosity (default 1)
```

### SEE ALSO

* [components-cli component-archive component-references](components-cli_component-archive_component-references.md)	 - command to modify component references of a component descriptor

