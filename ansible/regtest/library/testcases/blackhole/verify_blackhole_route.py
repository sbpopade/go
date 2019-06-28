
import shlex
from ansible.module_utils.basic import AnsibleModule
from collections import OrderedDict

DOCUMENTATION = """
---
module: verify_blackhole_route
author: Platina Systems
short_description: Module to verify blackhole addition to different route tables
description:
    Module to test and verify blackhole configurations and log the same.
options:
     switch_name:
      description:
        - Name of the switch on which tests will be performed.
      required: False
      type: str
    spine_list:
      description:
        - List of all spine switches.
      required: False
      type: list
      default: []
    leaf_list:
      description:
        - List of all leaf switches.
      required: False
      type: list
      default: []
    hash_name:
      description:
        - Name of the hash in which to store the result in redis.
      required: False
      type: str
    subnet_mask:
        description:
            - subnet value for which blackhole needs to be added
    eth_list:
        description
            - list of interfaces on which blackhole will be added
    log_dir_path:
      description:
        - Path to log directory where logs will be stored.
      required: False
      type: str
"""

EXAMPLES = """
- name: Verify black hole addition to tables
  verify_blackhole_route:
          switch_name: "{{ inventory_hostname }}"
          eth_list: "5"
          subnet_mask: "24"
          spine_list: "{{ groups['spine'][1] }}"
          leaf_list: "{{ groups['leaf'][1] }}"
          hash_name: "{{ hostvars['server_emulator']['hash_name'] }}"
          log_dir_path: "{{ blackhole_route_dir }}"
"""

RETURN = """
hash_dict:
  description: Dictionary containing key value pairs to store in hash.
  returned: always
  type: dict
"""

HASH_DICT = OrderedDict()
result_status = True

def run_cli(module, cli):
    """
    Method to execute the cli command on the target node(s) and
    returns the output.
    :param module: The Ansible module to fetch input parameters.
    :param cli: The complete cli string to be executed on the target node(s).
    :return: Output/Error or None depending upon the response from cli.
    """
    cli = shlex.split(cli)
    rc, out, err = module.run_command(cli)

    if out:
        return out.rstrip()
    elif err:
        return err.rstrip()
    else:
        return None


def execute_commands(module, cmd):
    """
    Method to execute given commands and return the output.
    :param module: The Ansible module to fetch input parameters.
    :param cmd: Command to execute.
    :return: Output of the commands.
    """
    global HASH_DICT

    if ('service' in cmd and 'restart' in cmd) or module.params['dry_run_mode']:
        out = None
    else:
        out = run_cli(module, cmd)

    # Store command prefixed with exec time as key and
    # command output as value in the hash dictionary
    exec_time = run_cli(module, 'date +%Y%m%d%T')
    key = '{0} {1} {2}'.format(module.params['switch_name'], exec_time, cmd)
    HASH_DICT[key] = out

    return out

def verify_blackhole_route_tables(module):

    global result_status
    failure_summary = ''

    eth_list = module.params['eth_list']
    subnet_mask = module.params['subnet_mask']
    leaf_list = module.params['leaf_list'][:]
    spine_list = module.params['spine_list'][:]
    switch_name = module.params['switch_name']

    if switch_name in leaf_list:
            p_list = spine_list
    elif switch_name in spine_list:
            p_list = leaf_list

    if subnet_mask == "24":
        blackhole_ip = '10.0.{}.0/24'.format(eth_list)
        ip_check = '10.0.{0}.0'.format(eth_list)
    elif subnet_mask == "32":
        blackhole_ip = '10.0.{0}.{1}/32'.format(eth_list,p_list[0][-2:])
        ip_check = '10.0.{0}.{1}'.format(eth_list,p_list[0][-2:])

    #verify kernel ip route table
    out = execute_commands(module,'ip route')
    if ip_check not in out:
        result_status = False
        failure_summary += 'blackhole {} not added in kernel route table'.format(blackhole_ip)

    #verify goes table
    out = execute_commands(module,'goes vnet show ip fib')
    for line in out.splitlines():
        if blackhole_ip in line:
            if 'drop' not in line:
                result_status = False
                failure_summary += 'Drop not found in goES table for blackhole {}'.format(blackhole_ip)
                break
            break

    #verify tcam table(check in all four pipelines)
    out = execute_commands(module,'goes vnet show fe1 tcam')
    count = 0
    for line in out.splitlines():
        if ip_check in line:
            if 'Drop' not in line:
                result_status = False
                failure_summary+=' Adj:Drop not found in tcam table for blackhole {}'.format(blackhole_ip)
            count+=1

#    if count !=4:
#        result_status = False
#        failure_summary += ' Adj:Drop found in {} pipelines'.format(count)

    return failure_summary


#verify ping status
def verify_ping(module):

    switch_name = module.params['switch_name']
    eth_list = module.params['eth_list']
    leaf_list = module.params['leaf_list'][:]
    spine_list = module.params['spine_list'][:]

    global result_status, HASH_DICT
    packet_count = 5
    failure_summary = ''

    if switch_name in leaf_list:
            p_list = spine_list
    elif switch_name in spine_list:
            p_list = leaf_list

    cmd = "ping -c {3} -I 10.0.{0}.{1} 10.0.{0}.{2}".format(eth_list, switch_name[-2:], p_list[0][-2:],packet_count)

    ping_out = execute_commands(module, cmd)

    if '100% packet loss' not in ping_out:
            result_status = False
            failure_summary += 'Ping from switch {} to {}'.format(switch_name, p_list[0])
            failure_summary += ' are received in the output of '
            failure_summary += 'command {}\n'.format(cmd)

    return failure_summary

def main():

    module = AnsibleModule(
        argument_spec=dict(
                    switch_name=dict(required=False, type='str'),
                    spine_list=dict(required=False, type='list', default=[]),
                    leaf_list=dict(required=False, type='list', default=[]),
                    subnet_mask=dict(required=False, type='str'),
                    eth_list = dict(required=False, type='str', default=''),
                    hash_name=dict(required=False,type='str'),
                    log_dir_path=dict(required=False,type='str'),
                    dry_run_mode=dict(required=False, type='bool', default=False)
        )
    )

    global result_status,HASH_DICT
    if module.params['dry_run_mode']:
        cmds_list = []

        execute_commands(module, 'ip route delete 10.0.{}.0/24'.format(module.params['interface_name'][-1]))
        execute_commands(module, 'ip route sdd blackhole 10.0.{}.0/24'.format(module.params['interface_name'][-1]))
        execute_commands(module, 'ip route')
        execute_commands(module, 'goes vnet show ip fib')
        execute_commands(module, 'goes vnet show fe1 tcam')
        execute_commands(module, "ping -c {3} -I 10.0.{0}.{1} 10.0.{0}.{2}".format(module.params['interface_name'][-1],
                                              module.params['switch_name'][-2:], module.params['leaf_list'][0][-2:],5))
        execute_commands(module, 'redis-cli -h "{{ bmc_redis_ip }}" hset platina psu.powercycle true')
        execute_commands(module, 'goes status')

        for key, value in HASH_DICT.iteritems():
            cmds_list.append(key)

        # Exit the module and return the required JSON.
        module.exit_json(
            cmds=cmds_list
        )
    else:

        res1 = verify_blackhole_route_tables(module)
        res2 = verify_ping(module)
        HASH_DICT['result.detail'] = res1 + res2
        HASH_DICT['result.status'] = 'Passed' if result_status else 'Failed'
       
        # Create a log file
        log_file_path = module.params['log_dir_path']
        log_file_path += '/{}.log'.format(module.params['hash_name'])
        log_file = open(log_file_path, 'a')
        for key, value in HASH_DICT.iteritems():
            log_file.write(key)
            log_file.write('\n')
            log_file.write(str(value))
            log_file.write('\n')
            log_file.write('\n')

        log_file.close()

    # Exit the module and return the required JSON.
        module.exit_json(
        hash_dict=HASH_DICT,
        log_file_path=log_file_path
    )

if __name__ == '__main__':
        main()
