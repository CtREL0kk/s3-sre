#!/usr/bin/env python3
import logging
import os
import time
from pathlib import Path

import requests

API_URL = os.getenv('PROBE_URL') + '/api/v1'
LOGIN = os.environ['PROBE_LOGIN']
PASSWORD = os.environ['PROBE_PASSWORD']
INTERVAL = float(os.getenv('PROBE_INTERVAL', '10'))
TIMEOUT = 8
PROBE_FILE = Path(__file__).with_name('bigtxtfile.txt')


def api_request(session, method, path, **kwargs):
    response = session.request(method, API_URL + path, timeout=TIMEOUT, **kwargs)
    response.raise_for_status()
    return response


def delete_object(session, object_id):
    if object_id is None:
        return
    try:
        api_request(session, 'DELETE', f'/objects/{object_id}')
    except requests.RequestException:
        logging.warning('Could not delete probe object %s', object_id)


def check_upload_and_download(session, folder_id):
    file_id = None
    try:
        with PROBE_FILE.open('rb') as file:
            response = api_request(session, 'POST', f'/objects/{folder_id}/files', files={
                'file': (PROBE_FILE.name, file, 'text/plain'),
            })
        file_id = response.json()['id']

        response = api_request(session, 'GET', f'/objects/{file_id}/content')
        if response.content != PROBE_FILE.read_bytes():
            raise ValueError('Downloaded file does not match uploaded content')

        logging.info('Upload and download succeeded')
    finally:
        delete_object(session, file_id)


def main():
    logging.basicConfig(level=logging.INFO, format='%(asctime)s %(levelname)s %(message)s')
    folder_id = None
    while True:
        try:
            with requests.Session() as session:
                response = api_request(session, 'POST', '/auth/login', json={
                    'login': LOGIN,
                    'password': PASSWORD,
                })
                session.headers['Authorization'] = 'Bearer ' + response.json()['access_token']

                if folder_id is None:
                    response = api_request(session, 'POST', '/objects', json={
                        'name': f'probe-{time.time_ns()}',
                    })
                    folder_id = response.json()['id']
                    logging.info('Created probe folder %s', folder_id)

                check_upload_and_download(session, folder_id)
        except (requests.RequestException, ValueError, KeyError) as error:
            logging.error('Probe failed: %s', error)
        time.sleep(INTERVAL)


if __name__ == '__main__':
    main()
