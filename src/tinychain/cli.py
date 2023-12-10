# A simple CLI client for the tinychain.
import argparse

def node_command(args):
    print("Node command executed with arguments:", args)

def call_command(args):
    print("Call command executed with arguments:", args)

def send_command(args):
    print("Send command executed with arguments:", args)

def main():
    parser = argparse.ArgumentParser(
        prog='tinychain',  # Custom command name
        usage='%(prog)s [options] <command> [<args>]',
        description='The tiny smart contract blockchain',  # Custom description
        epilog='><'  # Custom epilog
    )
    subparsers = parser.add_subparsers(title="Subcommands", dest="subcommand")

    # Node subcommand
    node_parser = subparsers.add_parser("node", help="Run a tinychain node")
    node_parser.add_argument("node_arg", help="Node command argument")
    node_parser.set_defaults(func=node_command)

    # Call subcommand
    call_parser = subparsers.add_parser("call", help="Read the tinychain")
    call_parser.add_argument("call_arg", help="Call command argument")
    call_parser.set_defaults(func=call_command)

    # Send subcommand
    send_parser = subparsers.add_parser("send", help="Write the tinychain")
    send_parser.add_argument("send_arg", help="Send command argument")
    send_parser.set_defaults(func=send_command)

    args = parser.parse_args()
    if not vars(args) or not hasattr(args, 'func'):
        parser.print_help()
    else:
        args.func(args)


if __name__ == "__main__":
    main()
