export interface Contact {
  id: string;
  name: string;
  email: string;
  phone?: string;
  company?: string;
  created_at?: string;
}

export interface Deal {
  id: string;
  title: string;
  value: number;
  stage: string;
  contact_id?: string;
  contact_name?: string;
}

export interface Task {
  id: string;
  title: string;
  done: boolean;
  due_date?: string;
}

export interface DashboardSummary {
  total_contacts: number;
  open_deals: number;
  pipeline_value: number;
  tasks_due: number;
  deals_by_stage?: { stage: string; count: number; value: number }[];
  contacts_per_month?: { month: string; count: number }[];
}

export interface AuditLog {
  id: string;
  action: string;
  entity: string;
  entity_id?: string;
  created_at?: string;
}

export const DEAL_STAGES = [
  "lead",
  "qualified",
  "proposal",
  "negotiation",
  "won",
  "lost",
] as const;
