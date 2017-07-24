FROM node:{{ .Version }}

COPY package.json /app
RUN npm install && npm cache clean

COPY . /app
#RUN chown smith: /app/ -R

#USER smith

EXPOSE 5000
ENTRYPOINT ["node"]

