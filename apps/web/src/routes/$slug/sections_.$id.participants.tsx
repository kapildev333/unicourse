import { createFileRoute } from "@tanstack/react-router";
import { ParticipantsPage } from "../../features/section/participants";

export const Route = createFileRoute("/$slug/sections_/$id/participants")({
  component: function Page() {
    const { slug, id } = Route.useParams();
    return <ParticipantsPage slug={slug} sectionId={id} />;
  },
});
