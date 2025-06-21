import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import { z } from "zod";

// MLS (Major League Soccer) league id
const LEAGUE_IDS: Record<string, number> = {
	MLS: 253,
} as const;

export type OK<Data> = {
	ok: true;
	value: Data;
	error: null;
};
export function ok<Data>(data: Data): OK<Data> {
	return {
		ok: true,
		get value() {
			return data;
		},
		error: null,
	};
}

export type NotOK<Error> = { ok: false; error: Error };
export function err<E>(error: E): NotOK<E> {
	return {
		ok: false,
		error,
	};
}

export type Result<Data, Error = null> = OK<Data> | NotOK<Error>;

const headers = {
	"x-rapidapi-key": "91be9b12c36d01fd71847355d020c8d7",
	Accept: "application/json",
};

async function makeRequest<T>(
	path: string,
	params: Record<string, any>,
): Promise<Result<T, string>> {
	try {
		const response = await fetch(
			`https://v3.football.api-sports.io${path}?${new URLSearchParams(
				params,
			).toString()}`,
			{
				method: "GET",
				headers,
			},
		);
		if (!response.ok) {
			return err(`HTTP error! status: ${response.status}`);
		}
		return ok((await response.json()) as T);
	} catch (error) {
		console.error("Error making request:", error);
		return err(`${error}`);
	}
}

const server = new McpServer({
	name: "maestro",
	version: "1.0.0",
	capabilities: {
		resources: {},
		tools: {},
	},
});

type TeamsResponse = {
	parameters: {
		league: string;
		season: string;
	};
	errors: [];
	results: number;
	response: [
		{
			team: {
				id: number;
				name: string;
				code: string;
				country: string;
				founded: number;
				national: boolean;
				logo: string;
			};
		},
	];
};

server.tool(
	"get-teams",
	"Get a list of teams in a league",
	{
		league: z.enum(["MLS"]),
		season: z.number().describe("The season year (YYYY)"),
	},
	async ({ league, season }) => {
		const result = await makeRequest<TeamsResponse>("/teams", {
			league: LEAGUE_IDS[league],
			season,
		});
		if (!result.ok) {
			return {
				content: [
					{
						type: "text",
						text: `Failed to fetch teams: ${result.error}`,
					},
				],
			};
		}

		return {
			content: [
				{
					type: "text",
					text: JSON.stringify(result.value),
				},
			],
		};
	},
);

server.tool(
	"get-team-fixtures",
	"Get all fixtures for a team in a season",
	{
		team: z.number().describe("The team id"),
		season: z.number().describe("The season year (YYYY)"),
	},
	async ({ team, season }) => {
		const result = await makeRequest("/fixtures", {
			team,
			season,
		});
		if (!result.ok) {
			return {
				content: [
					{
						type: "text",
						text: `Failed to fetch fixtures: ${result.error}`,
					},
				],
			};
		}

		return {
			content: [
				{
					type: "text",
					text: JSON.stringify(result.value),
				},
			],
		};
	},
);

type FixturesResponse = {
	results: number;
	response: [
		{
			fixture: {
				id: number;
				referee: string | null;
				timezone: string;
				date: string;
				timestamp: number;
				periods: {
					first: number | null;
					second: number | null;
				};
				venue: {
					id: number;
					name: string;
					city: string;
				};
				status: {
					long: string;
					short: string;
					elapsed: number;
					extra: number | null;
				};
			};
			league: {
				id: number;
				name: string;
				country: string;
				logo: string;
				flag: string;
				season: number;
				round: string;
			};
			teams: {
				home: {
					id: number;
					name: string;
					logo: string;
					winner: boolean;
				};
				away: {
					id: number;
					name: string;
					logo: string;
					winner: boolean;
				};
			};
			goals: {
				home: number;
				away: number;
			};
			score: {
				halftime: {
					home: number;
					away: number;
				};
				fulltime: {
					home: number | null;
					away: number | null;
				};
				extratime: {
					home: number | null;
					away: number | null;
				};
				penalty: {
					home: number | null;
					away: number | null;
				};
			};
		},
	];
};
server.tool(
	"get-goal-stats",
	"Get goal statistics for a team in a season. Includes total goals scored and conceded and averages per game.",
	{
		team: z.number().describe("The team id"),
		season: z.number().describe("The season year (YYYY)"),
	},
	async ({ team, season }) => {
		const fixturesResult = await makeRequest<FixturesResponse>("/fixtures", {
			team,
			season,
		});
		if (!fixturesResult.ok) {
			return {
				content: [
					{
						type: "text",
						text: `Failed to fetch fixtures: ${fixturesResult.error}`,
					},
				],
			};
		}

		let goalScored = 0;
		let goalConceded = 0;
		let cleanSheets = 0;

		for (const fixture of fixturesResult.value.response) {
			// Check if the team is playing at home or away
			if (fixture.teams.home.id === team) {
				// Team is playing at home
				goalScored += fixture.goals.home;
				goalConceded += fixture.goals.away;
				if (fixture.goals.home === 0) {
					cleanSheets++;
				}
			} else if (fixture.teams.away.id === team) {
				// Team is playing away
				goalScored += fixture.goals.away;
				goalConceded += fixture.goals.home;
				if (fixture.goals.away === 0) {
					cleanSheets++;
				}
			}
		}

		const averageGoalsScored = goalScored / fixturesResult.value.results;
		const averageGoalsConceded = goalConceded / fixturesResult.value.results;
		const percentageCleanSheets = cleanSheets / fixturesResult.value.results;

		return {
			content: [
				{
					type: "text",
					text: `The team has scored ${goalScored} goals and conceded ${goalConceded} goals in ${
						fixturesResult.value.results
					} games. Their average goals scored per game is ${averageGoalsScored.toFixed(
						2,
					)} and their average goals conceded per game is ${averageGoalsConceded.toFixed(
						2,
					)}. Their percentage of clean sheets is ${percentageCleanSheets.toFixed(
						2,
					)}%.`,
				},
			],
		};
	},
);

const transport = new StdioServerTransport();
await server.connect(transport);

console.error("MCP Soccer Statistics Server running");
