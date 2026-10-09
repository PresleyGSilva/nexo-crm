import type { AuditLog, Contact, DashboardSummary, Deal, Task } from "./types";

export const MOCK_CONTACTS: Contact[] = Array.from({ length: 20 }, (_, i) => ({
  id: `mock-contact-${i + 1}`,
  name: `Contato Demonstração ${i + 1}`,
  email: `contato${i + 1}@exemplo.com`,
  phone: `+55 11 9${String(8000 + i)}-${String(1000 + i).slice(0, 4)}`,
  company: `Empresa ${((i % 5) + 1)}`,
}));

const STAGES = ["lead", "qualified", "proposal", "negotiation", "won", "lost"];

export const MOCK_DEALS: Deal[] = Array.from({ length: 15 }, (_, i) => ({
  id: `mock-deal-${i + 1}`,
  title: `Negócio ${i + 1}`,
  value: (i + 1) * 2500,
  stage: STAGES[i % STAGES.length],
  contact_name: `Contato Demonstração ${(i % 20) + 1}`,
}));

export const MOCK_TASKS: Task[] = Array.from({ length: 6 }, (_, i) => ({
  id: `mock-task-${i + 1}`,
  title: `Tarefa ${i + 1}: follow-up comercial`,
  done: i % 3 === 0,
  due_date: new Date(Date.now() + i * 86400000).toISOString(),
}));

export const MOCK_SUMMARY: DashboardSummary = {
  total_contacts: 20,
  open_deals: 11,
  pipeline_value: 187500,
  tasks_due: 4,
  deals_by_stage: [
    { stage: "lead", count: 3, value: 15000 },
    { stage: "qualified", count: 3, value: 30000 },
    { stage: "proposal", count: 2, value: 35000 },
    { stage: "negotiation", count: 3, value: 67500 },
    { stage: "won", count: 2, value: 25000 },
    { stage: "lost", count: 2, value: 15000 },
  ],
  contacts_per_month: [
    { month: "Jun", count: 2 },
    { month: "Jul", count: 3 },
    { month: "Ago", count: 4 },
    { month: "Set", count: 5 },
    { month: "Out", count: 6 },
  ],
};

export const MOCK_AUDIT: AuditLog[] = Array.from({ length: 8 }, (_, i) => ({
  id: `mock-audit-${i + 1}`,
  action: ["create", "update", "stage_change"][i % 3],
  entity: ["contact", "deal", "task"][i % 3],
  entity_id: `mock-${i + 1}`,
  created_at: new Date(Date.now() - i * 3600000).toISOString(),
}));
