package recognition

import "github.com/atlan/mibo-media-server/internal/database"

type GraphConstructInput = ManifestBuildInput

type GraphConstructOutput struct {
	ManifestScope             ManifestScope
	MediaGraphNodes           []database.MediaGraphNode
	MediaGraphEdges           []database.MediaGraphEdge
	MediaGraphClassifications []database.MediaGraphClassification
	Candidates                []database.RecognitionCandidate
	Evidence                  []database.RecognitionEvidence
}

func ConstructGraphFromInventory(input GraphConstructInput) GraphConstructOutput {
	graph := buildMediaGraphFromInventory(input)
	output := constructManifestOutputFromGraph(graph, input)
	return GraphConstructOutput{
		ManifestScope:             output.ManifestScope,
		MediaGraphNodes:           graphNodesFromMediaGraph(graph),
		MediaGraphEdges:           graphEdgesFromMediaGraph(graph),
		MediaGraphClassifications: graphClassificationsFromMediaGraph(graph),
		Candidates:                output.Candidates,
		Evidence:                  output.Evidence,
	}
}
