package main

import (
	"time"

	"github.com/BMokarzel/rhaxis-code.git/domain"
)

// View helpers: convertem tipos do domain (com interfaces e nomes de campo
// idiomáticos ao Go) em map[string]any JSON-friendly para o cliente HTTP.
// Ficam aqui, não no domain, porque o domain deve ser agnóstico de transporte.

func nodeView(n domain.Node) map[string]any {
	if n == nil {
		return nil
	}
	v := map[string]any{
		"urn":      n.URN(),
		"kind":     n.Kind(),
		"name":     n.Name(),
		"resolved": n.Resolved(),
	}
	// Display hints derivados do Kind — calculados na view (não
	// persistidos). Se vazios, omitimos: o frontend cai no default.
	if s := domain.DisplayShapeFor(n.Kind()); s != "" {
		v["displayShape"] = s
	}
	if c := domain.DisplayCategoryFor(n.Kind()); c != "" {
		v["displayCategory"] = c
	}
	if lang := n.Language(); lang != "" {
		v["language"] = lang
	}
	if svc := n.ServiceURN(); svc != "" {
		v["serviceURN"] = svc
	}

	// Campos kind-specific — type switch dedicado para não expor os tipos
	// internos direto (evita "URNValue" etc no wire).
	switch t := n.(type) {
	case *domain.Service:
		v["framework"] = t.Framework
		v["runtime"] = t.Runtime
		v["repoURL"] = t.RepoURL
		if !t.LastExtractedAt.IsZero() {
			v["lastExtractedAt"] = t.LastExtractedAt.Format(time.RFC3339)
		}
		if t.SourceRev != "" {
			v["sourceRev"] = t.SourceRev
		}
	case *domain.Database:
		v["engine"] = t.Engine
	case *domain.Broker:
		v["engine"] = t.Engine
	case *domain.Endpoint:
		v["httpMethod"] = t.HTTPMethod
		v["pathTemplate"] = t.PathTemplate
		v["framework"] = t.Framework
		if t.HandlerURN != "" {
			v["handlerURN"] = t.HandlerURN
		}
	case *domain.Function:
		v["signature"] = t.Signature
		v["isAsync"] = t.IsAsync
	case *domain.Method:
		v["signature"] = t.Signature
		v["isAsync"] = t.IsAsync
		if t.OwnerTypeURN != "" {
			v["ownerTypeURN"] = t.OwnerTypeURN
		}
	case *domain.IfNode:
		v["conditionText"] = t.ConditionText
	case *domain.SwitchNode:
		v["discriminant"] = t.Discriminant
	case *domain.LoopNode:
		v["loopKind"] = t.Kind_
	case *domain.CallFunction:
		if t.TargetURN != "" {
			v["targetURN"] = t.TargetURN
		}
	case *domain.CallHTTP:
		v["httpMethod"] = t.HTTPMethod
		v["pathTemplate"] = t.PathTemplate
		v["targetHint"] = t.TargetHint
		if t.TargetURN != nil {
			v["targetURN"] = *t.TargetURN
		}
	case *domain.CallDB:
		v["operation"] = t.Operation
		if t.TargetURN != "" {
			v["targetURN"] = t.TargetURN
		}
	case *domain.PublishEvent:
		v["topic"] = t.Topic
		if t.TargetURN != "" {
			v["targetURN"] = t.TargetURN
		}
	case *domain.ConsumeEvent:
		v["topic"] = t.Topic
		if t.TargetURN != "" {
			v["targetURN"] = t.TargetURN
		}
	}
	return v
}

func flowNodeView(f domain.FlowNode) map[string]any {
	v := map[string]any{
		"node": nodeView(f.Node),
	}
	if len(f.Children) > 0 {
		kids := make([]map[string]any, len(f.Children))
		for i, c := range f.Children {
			kids[i] = flowNodeView(c)
		}
		v["children"] = kids
	}
	if len(f.Branches) > 0 {
		br := make(map[string][]map[string]any, len(f.Branches))
		for label, items := range f.Branches {
			arr := make([]map[string]any, len(items))
			for i, it := range items {
				arr[i] = flowNodeView(it)
			}
			br[label] = arr
		}
		v["branches"] = br
	}
	if f.Expansion != nil {
		v["expansion"] = map[string]any{
			"targetURN":      f.Expansion.TargetURN,
			"targetKind":     f.Expansion.TargetKind,
			"targetResolved": f.Expansion.TargetResolved,
		}
	}
	return v
}

func serviceMapView(sm domain.ServiceMap) map[string]any {
	services := make([]map[string]any, len(sm.Services))
	for i, s := range sm.Services {
		svc := s // pin so &svc is stable
		services[i] = nodeView(&svc)
	}
	ext := make([]map[string]any, len(sm.ExternalSystems))
	for i, e := range sm.ExternalSystems {
		ext[i] = nodeView(e)
	}
	deps := make([]map[string]any, len(sm.Dependencies))
	for i, d := range sm.Dependencies {
		deps[i] = map[string]any{
			"from":   d.From,
			"to":     d.To,
			"via":    d.Via,
			"weight": d.Weight,
		}
	}
	return map[string]any{
		"services":        services,
		"externalSystems": ext,
		"dependencies":    deps,
	}
}

func endpointListView(el domain.EndpointList) map[string]any {
	eps := make([]map[string]any, len(el.Endpoints))
	for i, e := range el.Endpoints {
		ep := e
		eps[i] = nodeView(&ep)
	}
	svc := el.Service
	return map[string]any{
		"service":   nodeView(&svc),
		"endpoints": eps,
	}
}

func endpointFlowView(ef domain.EndpointFlow) map[string]any {
	ep := ef.Endpoint
	return map[string]any{
		"endpoint": nodeView(&ep),
		"root":     flowNodeView(ef.Root),
	}
}
