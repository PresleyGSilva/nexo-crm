-- 003_seed_demo.sql — Idempotent demo seed: 1 org, 1 user, 20 contacts,
-- 15 deals, plus tasks/interactions/settings. Safe to re-run.
-- Demo login: demo@nexo.local / Demo1234!

-- Demo org + user + owner membership.
INSERT INTO orgs (id, name, slug) VALUES
  ('11111111-1111-1111-1111-111111111111', 'Acme Demo', 'acme-demo')
ON CONFLICT (id) DO NOTHING;

INSERT INTO users (id, name, email, password_hash) VALUES
  ('22222222-2222-2222-2222-222222222222', 'Demo User', 'demo@nexo.local', '$2a$10$r0UOGzGp6qwWpaQRsRHQBemB4jv/Ykb2n3z5pCYMIjLyY6qrs0MQq')
ON CONFLICT (id) DO NOTHING;

INSERT INTO memberships (user_id, org_id, role) VALUES
  ('22222222-2222-2222-2222-222222222222', '11111111-1111-1111-1111-111111111111', 'owner')
ON CONFLICT DO NOTHING;

INSERT INTO org_settings (org_id, settings) VALUES
  ('11111111-1111-1111-1111-111111111111',
   '{"currency":"BRL","theme":"light","pipeline_stages":["prospecting","qualified","proposal","negotiation","won","lost"]}')
ON CONFLICT (org_id) DO NOTHING;

-- 20 fictional contacts with deterministic ids.
WITH seed(name, email, phone, company, position, status, n) AS (
  SELECT * FROM (VALUES
    ('Ana Souza','ana.souza@vetor.com.br','+55 11 98711-0001','Vetor S.A.','CEO','active',1),
    ('Bruno Lima','bruno.lima@litorallog.com','+55 21 98711-0002','Litoral Log','Gerente Comercial','active',2),
    ('Carla Mendes','carla.mendes@prismasaude.com','+55 31 98711-0003','Prisma Saúde','Diretora de Marketing','active',3),
    ('Diego Alves','diego.alves@atlasenergia.com','+55 41 98711-0004','Atlas Energia','Founder','lead',4),
    ('Elisa Rocha','elisa.rocha@fariatech.com','+55 51 98711-0005','Faria Tech','CTO','active',5),
    ('Fabio Costa','fabio.costa@nobre.com','+55 61 98711-0006','Nobre Alimentos','Head de Operações','active',6),
    ('Gabi Nunes','gabi.nunes@kayamoda.com','+55 71 98711-0007','Kaya Moda','Sócia','lead',7),
    ('Henrique Dias','henrique.dias@portodigital.com','+55 81 98711-0008','Porto Digital','Diretor Financeiro','active',8),
    ('Igor Martins','igor.martins@sombra.games','+55 11 98711-0009','Sombra Games','CEO','active',9),
    ('Juliana Castro','juliana.castro@lume.adv.br','+55 21 98711-0010','Lume Advocacia','Advogada','active',10),
    ('Kaique Barbosa','kaique.barbosa@teto.com','+55 31 98711-0011','Teto Construtora','Engenheiro','lead',11),
    ('Larissa Pinto','larissa.pinto@brisaviagens.com','+55 41 98711-0012','Brisa Viagens','Coordenadora','active',12),
    ('Marcos Vieira','marcos.vieira@nexobank.com','+55 51 98711-0013','Nexo Bank','Gerente de Contas','active',13),
    ('Natalia Ramos','natalia.ramos@ondina.com','+55 61 98711-0014','Ondina Cosméticos','Diretora Comercial','active',14),
    ('Otavio Freire','otavio.freire@faroledu.com','+55 71 98711-0015','Farol Educação','Coordenador','inactive',15),
    ('Paula Teixeira','paula.teixeira@trilha.com','+55 81 98711-0016','Trilha Sports','Supervisora','active',16),
    ('Rafael Cardoso','rafael.cardoso@melodia.com','+55 11 98711-0017','Melodia Eventos','Produtor','lead',17),
    ('Sofia Almeida','sofia.almeida@graocafe.com','+55 21 98711-0018','Grão Café','Barista-chefe','active',18),
    ('Thiago Moreira','thiago.moreira@velaseguros.com','+55 31 98711-0019','Vela Seguros','Corretor','active',19),
    ('Vanessa Correia','vanessa.correia@pampaagro.com','+55 41 98711-0020','Pampa Agro','Agrônoma','active',20)
  ) AS v(name, email, phone, company, position, status, n)
)
INSERT INTO contacts (id, org_id, name, email, phone, company, position, status, created_by)
SELECT ('30000000-0000-0000-0000-' || lpad(n::text, 12, '0'))::uuid,
       '11111111-1111-1111-1111-111111111111',
       name, email, phone, company, position, status,
       '22222222-2222-2222-2222-222222222222'
FROM seed
ON CONFLICT (id) DO NOTHING;

-- 15 fictional deals across every pipeline stage.
WITH seed(title, contact_n, value_cents, stage, n) AS (
  SELECT * FROM (VALUES
    ('Contrato anual — Vetor S.A.',1,12000000,'prospecting',1),
    ('Licenças — Prisma Saúde',3,8500000,'qualified',2),
    ('Expansão — Atlas Energia',4,22000000,'proposal',3),
    ('Onboarding — Faria Tech',5,5400000,'negotiation',4),
    ('Renovação — Nobre Alimentos',6,9800000,'won',5),
    ('Piloto — Kaya Moda',7,2100000,'prospecting',6),
    ('Integração — Porto Digital',8,6700000,'qualified',7),
    ('Migração — Sombra Games',9,13400000,'proposal',8),
    ('Suporte premium — Lume Advocacia',10,3200000,'negotiation',9),
    ('Obra — Teto Construtora',11,18700000,'prospecting',10),
    ('Pacote corporativo — Brisa Viagens',12,4500000,'won',11),
    ('Conta PJ — Nexo Bank',13,7600000,'lost',12),
    ('Licenças — Farol Educação',15,3900000,'qualified',13),
    ('Patrocínio — Trilha Sports',16,2800000,'proposal',14),
    ('Apólice — Vela Seguros',19,9100000,'negotiation',15)
  ) AS v(title, contact_n, value_cents, stage, n)
)
INSERT INTO deals (id, org_id, contact_id, title, value_cents, stage, expected_close, created_by)
SELECT ('40000000-0000-0000-0000-' || lpad(n::text, 12, '0'))::uuid,
       '11111111-1111-1111-1111-111111111111',
       ('30000000-0000-0000-0000-' || lpad(contact_n::text, 12, '0'))::uuid,
       title, value_cents, stage,
       (CURRENT_DATE + (n || ' days')::interval)::date,
       '22222222-2222-2222-2222-222222222222'
FROM seed
ON CONFLICT (id) DO NOTHING;

-- Sample tasks.
INSERT INTO tasks (org_id, title, description, status, due_date, contact_id, created_by) VALUES
  ('11111111-1111-1111-1111-111111111111','Ligar para Ana Souza','Apresentar proposta anual','pending',CURRENT_DATE + 1,'30000000-0000-0000-0000-000000000001','22222222-2222-2222-2222-222222222222'),
  ('11111111-1111-1111-1111-111111111111','Enviar contrato — Nobre Alimentos','Minuta + SLA','in_progress',CURRENT_DATE + 2,'30000000-0000-0000-0000-000000000006','22222222-2222-2222-2222-222222222222'),
  ('11111111-1111-1111-1111-111111111111','Follow-up Atlas Energia','Retomar proposta técnica','pending',CURRENT_DATE + 3,'30000000-0000-0000-0000-000000000004','22222222-2222-2222-2222-222222222222'),
  ('11111111-1111-1111-1111-111111111111','Preparar demo — Porto Digital','Ambiente + dados fictícios','done',CURRENT_DATE - 1,'30000000-0000-0000-0000-000000000008','22222222-2222-2222-2222-222222222222'),
  ('11111111-1111-1111-1111-111111111111','Revisar apólice — Vela Seguros','Conferir coberturas','pending',CURRENT_DATE + 5,'30000000-0000-0000-0000-000000000019','22222222-2222-2222-2222-222222222222'),
  ('11111111-1111-1111-1111-111111111111','Agendar visita — Teto Construtora','Levar catálogo impresso','cancelled',CURRENT_DATE - 2,'30000000-0000-0000-0000-000000000011','22222222-2222-2222-2222-222222222222')
ON CONFLICT DO NOTHING;

-- Sample interactions.
INSERT INTO interactions (org_id, contact_id, deal_id, type, content, created_by) VALUES
  ('11111111-1111-1111-1111-111111111111','30000000-0000-0000-0000-000000000001','40000000-0000-0000-0000-000000000001','call','Primeira ligação: interesse alto, pediu proposta formal.','22222222-2222-2222-2222-222222222222'),
  ('11111111-1111-1111-1111-111111111111','30000000-0000-0000-0000-000000000003','40000000-0000-0000-0000-000000000002','email','Enviado deck comercial + tabela de preços.','22222222-2222-2222-2222-222222222222'),
  ('11111111-1111-1111-1111-111111111111','30000000-0000-0000-0000-000000000004','40000000-0000-0000-0000-000000000003','meeting','Reunião técnica: validaram escopo, falta aprovação do board.','22222222-2222-2222-2222-222222222222'),
  ('11111111-1111-1111-1111-111111111111','30000000-0000-0000-0000-000000000006','40000000-0000-0000-0000-000000000005','note','Cliente renovou por mais 12 meses.','22222222-2222-2222-2222-222222222222'),
  ('11111111-1111-1111-1111-111111111111','30000000-0000-0000-0000-000000000008','40000000-0000-0000-0000-000000000007','whatsapp','Confirmaram a demo para quinta-feira.','22222222-2222-2222-2222-222222222222'),
  ('11111111-1111-1111-1111-111111111111','30000000-0000-0000-0000-000000000013','40000000-0000-0000-0000-000000000012','email','Proposta recusada: preço acima do orçamento.','22222222-2222-2222-2222-222222222222'),
  ('11111111-1111-1111-1111-111111111111','30000000-0000-0000-0000-000000000011','40000000-0000-0000-0000-000000000010','call','Obra adiada para o próximo trimestre.','22222222-2222-2222-2222-222222222222'),
  ('11111111-1111-1111-1111-111111111111','30000000-0000-0000-0000-000000000019','40000000-0000-0000-0000-000000000015','meeting','Alinhamento de coberturas da apólice.','22222222-2222-2222-2222-222222222222')
ON CONFLICT DO NOTHING;

-- A couple of audit entries so the demo log is not empty.
INSERT INTO audit_logs (org_id, user_id, action, entity, entity_id, metadata)
SELECT '11111111-1111-1111-1111-111111111111',
       '22222222-2222-2222-2222-222222222222',
       'seed.applied', 'org', '11111111-1111-1111-1111-111111111111',
       '{"contacts":20,"deals":15}'::jsonb
WHERE NOT EXISTS (SELECT 1 FROM audit_logs WHERE action = 'seed.applied');
