import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import { z } from "zod";

// MLS (Major League Soccer) league id
const LEAGUE_IDS: Record<string, number> = {
	MLS: 253,
	"Club World Cup": 15,
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

function sanitize(params: Record<string, any>) {
	return Object.fromEntries(
		Object.entries(params).filter(([_, value]) => value != undefined),
	);
}

async function makeRequest<T>(
	path: string,
	params: Record<string, any>,
): Promise<Result<T, string>> {
	try {
		const response = await fetch(
			`https://v3.football.api-sports.io${path}?${new URLSearchParams(
				sanitize(params),
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
	"Get a list of teams in a league or competition",
	{
		league: z.enum(["MLS", "Club World Cup"]),
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
	"get-fixtures",
	"Search for fixtures in a season. At least a league or team must be provided.",
	{
		league: z.enum(["MLS", "Club World Cup"]).optional(),
		team: z.number().describe("The team id").optional(),
		season: z.number().describe("The season year (YYYY)"),
		upcoming: z.boolean().optional(),
		played: z.boolean().optional(),
	},
	async ({ league, team, season, upcoming, played }) => {
		const params: Record<string, any> = {
			season,
			league: league ? LEAGUE_IDS[league] : undefined,
			team: team ? team : undefined,
			status: upcoming ? "NS" : played ? "FT" : undefined,
		};
		const result = await makeRequest("/fixtures", params);
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

const predictionsDescription = `
  Get predictions about a fixture.

  The predictions are made using several algorithms including the poisson distribution, comparison of team statistics, last matches, players etc…

  Bookmakers odds are not used to make these predictions

  Also provides some comparative statistics between teams
  Available Predictions

  Match winner : Id of the team that can potentially win the fixture
  Win or Draw : If True indicates that the designated team can win or draw
  Under / Over : -1.5 / -2.5 / -3.5 / -4.5 / +1.5 / +2.5 / +3.5 / +4.5 *
  Goals Home : -1.5 / -2.5 / -3.5 / -4.5 *
  Goals Away -1.5 / -2.5 / -3.5 / -4.5 *
  Advice (Ex : Deportivo Santani or draws and -3.5 goals)
  * -1.5 means that there will be a maximum of 1.5 goals in the fixture, i.e : 1 goal
  `;

server.tool(
	"get-predictions",
	predictionsDescription,
	{ fixture: z.string().describe("The id of the fixture") },
	async ({ fixture }) => {
		const result = await makeRequest("/predictions", { fixture });
		if (!result.ok) {
			return {
				content: [
					{
						type: "text",
						text: `Failed to get predictions: ${result.error}`,
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
	"get-odds",
	predictionsDescription,
	{
		league: z.string().describe("The id of the league"),
		season: z.number().describe("The season year (YYYY)"),
		fixture: z.string().describe("The id of the fixture"),
	},
	async ({ fixture }) => {
		const result = await makeRequest("/odds", { fixture });
		if (!result.ok) {
			return {
				content: [
					{
						type: "text",
						text: `Failed to get odds: ${result.error}`,
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

const transport = new StdioServerTransport();
await server.connect(transport);

console.error("MCP Soccer Statistics Server running");
