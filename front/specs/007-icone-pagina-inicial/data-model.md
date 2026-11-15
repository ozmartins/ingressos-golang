# Data Model: Ícone para retornar à página inicial

Sem entidades, persistência ou mudança de dados. O único "estado" é a rota atual (já existente em `useRota`): o `LinkInicio` só lê `window.location.pathname` para saber se já está na raiz.
