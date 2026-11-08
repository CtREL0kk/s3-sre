FROM python:3.13-alpine
COPY requirements.txt /tmp/requirements.txt
RUN pip install --no-cache-dir -r /tmp/requirements.txt
COPY prober.py bigtxtfile.txt /probe/
ENV PYTHONUNBUFFERED=1
CMD ["python3", "/probe/prober.py"]
