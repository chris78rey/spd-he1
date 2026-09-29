# Folio

Folio valida el inicio de sesión contra Oracle desde el backend Go. Escribe el usuario y la contraseña personales de Oracle en la pantalla de acceso. Solo se permite el acceso a usuarios con el rol `SPD_EXTERNOS`, comprobado contra `USER_ROLE_PRIVS`. El navegador no conserva la contraseña y la conexión usada para validar el acceso se cierra al terminar esa comprobación. La sesión autoriza el acceso a Folio durante 8 horas.

Las variables `ORACLE_USER` y `ORACLE_PASSWORD` de `.env` son la cuenta técnica que el backend utiliza para consultar `planilla_digital`; no se usan para iniciar sesión. Esa cuenta necesita permiso `SELECT` sobre la tabla. Los datos del formulario se usan únicamente para validar las credenciales y comprobar el rol `SPD_EXTERNOS`. El servidor también necesita `ORACLE_HOST`, `ORACLE_PORT` y `ORACLE_SERVICE`. `ORACLE_SCHEMA` indica el propietario de `planilla_digital`; en el entorno de prueba es `DIGITALIZACION`.

## Desarrollo local

1. Copia y completa `.env` con el host, puerto y servicio Oracle.
2. En una terminal, ejecuta `npm run dev:api`.
3. En otra terminal, ejecuta `npm run dev` y abre la dirección que muestra Vite.

El frontend envía `/api` al backend local en el puerto `8080`. Puedes cambiar ese puerto con `API_PORT`; si lo haces, actualiza también el proxy de Vite. Para publicar Folio, sirve la aplicación y el backend bajo el mismo origen con HTTPS y configura `COOKIE_SECURE=true`.

La opción **Planilla digital** lee `planilla_digital` con la cuenta técnica de `.env` y presenta 10 filas por página, únicamente después de iniciar sesión con un usuario que tenga `SPD_EXTERNOS`. Los documentos de Folio siguen almacenándose en IndexedDB del navegador; no se migran a Oracle.

## Inicio automático en Linux

En un equipo con `systemd`, ejecuta `./scripts/install-service.sh` una vez. El script compila la interfaz y el backend, instala `folio.service` y lo habilita para iniciar en cada arranque. Luego abre `http://127.0.0.1:8080`. Los comandos de administración requieren `sudo`.
