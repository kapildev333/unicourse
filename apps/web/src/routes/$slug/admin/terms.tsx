import { createFileRoute } from "@tanstack/react-router";
import { TermsPage } from "../../../features/admin/terms";

function Page() {
  const { slug } = Route.useParams();
  return <TermsPage slug={slug} />;
}

export const Route = createFileRoute("/$slug/admin/terms")({ component: Page });
