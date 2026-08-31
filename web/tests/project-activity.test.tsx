import { render, screen } from "@testing-library/react";
import { ProjectActivity } from "@/components/project-activity";
import projectStatus from "@/public/data/project-status.json";

it("shows synchronized public GitHub activity without production claims", () => {
  render(<ProjectActivity />);
  expect(screen.getByText(/public github activity/i)).toBeInTheDocument();
  expect(screen.getByText(projectStatus.latestReleaseTag)).toBeInTheDocument();
  expect(screen.getByText(/does not mean a feature is production-ready/i)).toBeInTheDocument();
});
