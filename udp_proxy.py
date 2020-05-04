import sys
import time
import logging
import socket
import queue
import random
import concurrent.futures


class UDPClient:
    def __init__(self, ip: str, port: int):
        self.ip = ip
        self.port = port
        self.send_msg_queue = queue.Queue()
        self.receive_msg_queue = queue.Queue()

        # ramdom from 22223 to 22800
        self.proxy_port = random.randint(22223, 22800)
        self.proxy_socket = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        while True:
            try:
                self.proxy_socket.bind(('', self.proxy_port))
                # if not receive data in two mintue, raise timeout
                # self.proxy_socket.settimeout(120)
                self.proxy_socket.setblocking(False)
                logging.info("New client {}:{} use proxy port {}".format(
                    self.ip, self.port, self.proxy_port))
                break
            except socket.error as e:
                logging.error(
                    "port {} is already in used, use another one".format(self.proxy_port))
                logging.error(e)
                self.proxy_port = random.randint(30000, 50000)

    def __str__(self):
        return "UDPClient {}:{}".format(self.ip, self.port)

    def run(self, server_ip, server_port):
        start_time = time.time()
        while True:
            # logging.debug("client {}:{} queue size {} before send".format(self.ip, self.port, self.send_msg_queue.qsize()))
            try:
                msg = self.send_msg_queue.get(block=False)
                self.proxy_socket.sendto(msg, (server_ip, server_port))
                start_time = time.time()
                # logging.debug("send {} to server {}:{}".format(
                #     msg, server_ip, server_port))
            except queue.Empty as e:
                msg = None

            # receive message from server and put it into self receive_msg_queue
            try:
                data, addr = self.proxy_socket.recvfrom(32768)
                # logging.debug("receive {} from server {}:{}".format(
                #     data, addr[0], addr[1]))
                self.receive_msg_queue.put(data)
                start_time = time.time()
            except Exception as e:
                # logging.info(e)
                # logging.info("Client {}:{} use proxy port {} timeout!".format(
                #     self.ip, self.port, self.proxy_port))
                pass
            # timeout if not receive message from client and server after 120s
            if time.time() - start_time > 120:
                break
        return self.ip, self.port


class UDPClientManager:
    def __init__(self, num_proxy: int, local_port: int, remote_ip: str, remote_port: int):
        self.local_port = local_port
        self.remote_ip = remote_ip
        self.remote_port = remote_port
        self.num_proxy = num_proxy

        self.client_dict = {}
        self.threadExecutor = concurrent.futures.ThreadPoolExecutor(
            max_workers=num_proxy)

    def isClientExisted(self, ip: str, port: int):
        if (ip, port) in self.client_dict:
            return True
        else:
            return False

    def tryAddClient(self, ip: str, port: int):
        if self.isClientExisted(ip, port):
            # logging.debug("Client {}:{} already existed.".format(ip, port))
            return True
        elif len(self.client_dict) >= self.num_proxy:
            logging.debug(
                "Already have {} clients, ignore Client {}:{}".format(self.num_proxy, ip, port))
            return False
        else:
            client = UDPClient(ip, port)
            self.client_dict[(ip, port)] = client
            logging.info("Add Client {}:{}".format(ip, port))
            # add sub thread to proxy message in two queue
            future = self.threadExecutor.submit(
                client.run, self.remote_ip, self.remote_port)
            # delete client when finish
            future.add_done_callback(self.delClientAsync)
            return True

    def delClient(self, ip: str, port: int):
        del self.client_dict[(ip, port)]
        logging.info("Delete Client {}:{}".format(ip, port))

    def delClientAsync(self, future):
        ip, port = future.result()
        self.delClient(ip, port)

    def run(self):
        # main thread loop
        receive_from_clients_socket = socket.socket(
            socket.AF_INET, socket.SOCK_DGRAM)
        receive_from_clients_socket.bind(('', self.local_port))
        while True:
            # receive messages from clients and put it into correspond queue
            data, addr = receive_from_clients_socket.recvfrom(32768)
            # logging.debug("receive {} from client {}:{}".format(
            #     data, addr[0], addr[1]))
            if self.tryAddClient(addr[0], addr[1]):
                client = self.client_dict[(addr[0], addr[1])]
                client.send_msg_queue.put(data)
                # logging.debug("get Client {}".format(client))
                # logging.debug("client {}:{} queue size {} after put".format(addr[0], addr[1], client.send_msg_queue.qsize()))

            # get messages from queues and send it to correspond client
            for client in self.client_dict.values():
                while not client.receive_msg_queue.empty():
                    msg = client.receive_msg_queue.get()
                    receive_from_clients_socket.sendto(
                        msg, (client.ip, client.port))
                    # logging.debug("send {} to client {}:{}".format(
                    #     msg, client.ip, client.port))


if __name__ == "__main__":
    NUM_PROXY = 12
    FORMAT = '[%(levelname)s] [%(filename)s:%(lineno)d] %(message)s'
    logging.basicConfig(level=logging.DEBUG, format=FORMAT)

    if len(sys.argv) != 4:
        logging.critical(
            'Usage: python3 udp_proxy.py <local port> <remote ip> <remote port>')
        sys.exit(1)

    _, local_port, remote_ip, remote_port = sys.argv
    local_port, remote_port = int(local_port), int(remote_port)

    logging.debug("local_port: {}, remote_ip: {}, remote_port: {}".format(
        local_port, remote_ip, remote_port))

    manager = UDPClientManager(NUM_PROXY, local_port, remote_ip, remote_port)
    logging.info("begin proxy")
    manager.run()
