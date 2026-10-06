import { useQuery } from "@tanstack/react-query";
import { api, unwrap } from "../../api/client";

/** First 200 departments for pickers and name maps (ceiling: a search endpoint past 200). */
export function useAllDepartments(slug: string) {
  return useQuery({
    queryKey: [slug, "departments", "all"],
    queryFn: () =>
      unwrap(
        api.GET("/universities/{slug}/departments", {
          params: { path: { slug }, query: { limit: 200 } },
        }),
      ),
    staleTime: 60_000,
  });
}
