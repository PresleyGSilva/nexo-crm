import { Sidebar } from "@/components/Sidebar";
import { Topbar } from "@/components/ui";

export default function AppShell({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex min-h-screen">
      <Sidebar />
      <div className="flex min-w-0 flex-1 flex-col">
        <Topbar title={title} />
        <main className="flex-1 p-6">{children}</main>
      </div>
    </div>
  );
}
