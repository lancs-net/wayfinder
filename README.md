# wayfinder: OS Configuration Micro-Benchmarking Framework

Wayfinder is a generic OS performance evaluation platform.  Wayfinder is fully
automated and ensures both the accuracy and reproducibility of results, all the
while speeding up how fast tests are run on a system. Wayfinder is easily
extensible and offers convenient APIs to:

 1. Implement custom configuration space exploration techniques,
 2. Add new benchmarks; and,
 3. Support additional OS projects.

Wayfinder's capacity to automatically and efficiently explore a LibOS' can be
found in the [examples/](examples/) directory; as well as its ability to
efficiently isolate parallel experiments to avoid noisy neighbors.

## Configuration

New jobs are described using a configuration YAML file.

### Parameterization configuration

| Attribute   | Required | Definition                                                                                                 |
|-------------|----------|------------------------------------------------------------------------------------------------------------|
| `name`      | Yes      | The name of the variable.  This will be the same as the environmental argument passed to a `run` instance. |
| `type`      | Yes      | The variable type, one of: [`integer`, `string`].                                                          |
| `min`       | No       | Starting `integer` value.                                                                                  |
| `max`       | No       | Ending `integer` value.                                                                                    |
| `step`      | No       | How much to increment `integer` value by.  Default is `1`.                                                 |
| `step_mode` | No       | Whether to step by `increment` or by `power`.  When `power`, the `step_mode` is used as the base.          |
| `only`      | No       | Discrete list of values to vary the parameter by.                                                          |

#### Examples

1. Integer, min-max, static increment: `[1, 2, 3, 4, 5]`
   ```yaml
   params:
     - name: A
       type: integer
       min: 1
       max: 5
       step: 1
   ```

2. Integer, min-max, power increment: `[1, 2, 4, 8, 16]`
   ```yaml
   params:
     - name: B
       type: integer
       min: 1
       max: 16
       step: 2
       step_mode: power
   ```

3. Integer, fixed set: `[1, 20, 100]`
   ```yaml
   params:
     - name: C
       type: integer
       only: [1, 20, 100]
   ```

4. String, fixed set: `["hello", "world"]`
   ```yaml
   params:
     - name: D
       type: string
       only: ["hello", "world"]
   ```

When parameters A and B are used (seen above), the following permutation matrix
will be run via wayfinder:

|  # | `C`   | `D`     |
|----|-------|---------|
|  1 | `1`   | `hello` |
|  2 | `1`   | `world` |
|  3 | `20`  | `hello` |
|  4 | `20`  | `world` |
|  5 | `100` | `hello` |
|  6 | `100` | `world` |

### Runtime configuration

| Attribute      | Required | Description                                                             |
|----------------|----------|-------------------------------------------------------------------------|
| `name`         | Yes      | The name of the run.                                                    |
| `image`        | Yes      | Remote OCI image for the filesystem to use for the run.                 |
| `cmd`          | Yes      | The command to run within the OCI image during the run.                 |
| `devices`      | No       | List of additional devices to attach from the host to the run instance. |
| `cores`        | No       | Number of cores to allocate the run instance.  Default is `1`.          |
| `capabilities` | No       | List of capabilities the OCI filesystem should have access to.          |

All parameters defined in the YAML configuration are provided to `run`s as
environmental variables.  Every run directive can use a remote OCI image for
creating a flesystem with the needed dependencies of the action, for example:

#### Example 

```yaml
run:
  - name: build
    image: unikraft/kraft:staging
    cmd:
      |
      echo $C $D
```

Additional devices can be attached to a `run` directive or capabilities, for
example being able to manipulate the host network:

```yaml
run:
  - name: build
    image: unikraft/kraft:staging
    devices:
      - /dev/net/tun
    capabilities:
      - CAP_NET_ADMIN
    cmd:
      |
      brctl addbr test0
```

The cores which have been allocated from the host system to the runtime instance
via the scheduler are passed as environmental variables to the instance.  For
example, if `2` cores are required for the `run`, then they are passed like so:

```yaml
run:
  - name: build
    image: unikraft/kraft:staging
    cores: 2
    cmd:
      |
      taskset -c $WAYFINDER_CORE_ID0 ./path/to/executable1.sh &
      taskset -c $WAYFINDER_CORE_ID1 ./path/to/executable2.sh
```

This can be used by, for example, `taskset` to ensure isolation.

### Input and output artifacts

All permutations may need information passed into it from the host system or
artifacts are generated from the result of a permutation, such as a OS binary.
These I/O can also be specified in the YAML configuration.

#### Inputs

| Attribute     | Required | Description                                                              |
|---------------|----------|--------------------------------------------------------------------------|
| `source`      | Yes      | The source of the file on the host to place in the run instance.         |
| `destination` | Yes      | The destination of the file to place in OCI filesystem the run instance. |

#### Outputs

| Attribute | Required | Description                                                                            |
|-----------|----------|----------------------------------------------------------------------------------------|
| `path`    | Yes      | The location of an artifact in the OCI filesystem created during the instance runtime. |

#### Example

```yaml
inputs:
  # Use the same DNS entries as the host system within the runtime instance.
  - source: /etc/resolv.conf
    destination: /etc/resolv.conf

outputs:
  # Output artifacts from the runtime instance.
  - path: /path/to/binary
  - path: /results.txt
```

## Getting started and usage

To get started using wayfinder, download the [latest
release](https://github.com/lancs-net/wayfinder/releases) and install on your
host system.  Once installed, you can use `wayfinder` as a CLI program:

```
Run a specific experiment job.

Usage:
  wayfinder run [OPTIONS...] [FILE]

Flags:
  -O, --allow-override            Override contents in directories (otherwise tasks allowed to fail).
  -b, --bridge string              (default "wayfinder0")
      --cpu-sets string           Specify which CPUs to run experiments on. (default "2-48")
  -D, --dry-run                   Run without affecting the host or running the jobs.
  -h, --help                      help for run
  -n, --hostnet string             (default "eth0")
  -r, --max-retries int           Maximum number of retries for a run.
  -g, --schedule-grace-time int   Number of seconds to gracefully wait in the scheduler. (default 1)
  -s, --subnet string              (default "172.88.0.1/16")
  -w, --workdir string            Specify working directory for outputting results, data, file systems, etc.

Global Flags:
  -v, --verbose   Enable verbose logging
```

Example configuration files can be found in [examples/](examples/) directory of
this repository.

## Cite

```bibtex
@inproceedings{Jung2021,
  title     = {Wayfinder: Towards Automatically Deriving Optimal OS Configurations},
  author    = {Jung, Alexander and Lefeuvre, Hugo           and Rotsos, Charalampos, and
               Pierre, Olivier and O\~{n}oro-Rubio, Daniel, and Niepert, Mathias,    and
               Huici, Felipe},
  journal   = {12th ACM SIGOPS Asia-Pacific Workshop on Systems},
  year      = {2021},
  series    = {APSys'21},
  publisher = {ACM},
  address   = {New York, NY, USA},
  doi       = {10.1145/3476886.3477506},
  isbn      = {978-1-4503-8698-2/21/08}
}
```

### Resources

 * [Paper](https://dl.acm.org/doi/10.1145/3476886.3477506) ([pdf](https://dl.acm.org/doi/pdf/10.1145/3476886.3477506))
 * [Video](https://youtu.be/YLf86gcHW4E)

## License

Wayfinder is licensed under `BSD-3-Clause`.  Read more in
[`LICENSE.md`](/LICENSE.md).
