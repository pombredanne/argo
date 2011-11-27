package argo

import (
	"fmt"
	"io"
	"json"
	"os"
	"strings"
)

type Graph struct {
	store Store
	prefixes map[string]string
}

func CreateGraph(store Store) (graph *Graph) {
	graph = &Graph{store: store}
	graph.init()

	return graph
}

func CreateGraphWithMemoryStore() (graph *Graph) {
	graph = &Graph{store: Store(CreateMemoryStore())}
	graph.init()

	return graph
}

func (graph *Graph) init() {
	graph.prefixes = map[string]string{"http://www.w3.org/1999/02/22-rdf-syntax-ns#": "rdf"}
}

func (graph *Graph) Bind(uri string, prefix string) (ns *Namespace) {
	graph.prefixes[uri] = prefix
	return CreateNamespace(uri)
}

func (graph *Graph) LookupAndBind(prefix string) (ns *Namespace, err os.Error) {
	uri, err := Lookup(prefix)
	if err == nil {
		return graph.Bind(uri, prefix), nil
	}
	
	return nil, err
}

func (graph *Graph) Store() (store Store) {
	return graph.store
}

func (graph *Graph) SetStore(store Store) {
	graph.store = store
}

func (graph *Graph) Add(triple *Triple) {
	graph.store.Add(triple)
}

func (graph *Graph) AddTriple(subject *Term, predicate *Term, object *Term) {
	graph.store.Add(CreateTriple(subject, predicate, object))
}

func (graph *Graph) AddQuad(subject *Term, predicate *Term, object *Term, context *Term) {
	graph.store.Add(CreateQuad(subject, predicate, object, context))
}

func (graph *Graph) Remove(triple *Triple) {
	graph.store.Remove(triple)
}

func (graph *Graph) RemoveTriple(subject *Term, predicate *Term, object *Term) {
	graph.store.Remove(CreateTriple(subject, predicate, object))
}

func (graph *Graph) RemoveQuad(subject *Term, predicate *Term, object *Term, context *Term) {
	graph.store.Remove(CreateQuad(subject, predicate, object, context))
}

func (graph *Graph) Clear() {
	graph.store.Clear()
}

func (graph *Graph) Triples() (triples []*Triple) {
	triples = []*Triple{}

	graph.store.ResetIter()

	for {
		if triple, ok := graph.store.Next(); ok {
			triples = append(triples, triple)
		} else {
			break
		}
	}

	return triples
}

func (graph *Graph) TriplesBySubject() (subjects map[*Term][]*Triple) {
	subjects = make(map[*Term][]*Triple)

	graph.store.ResetIter()

	for {
		if triple, ok := graph.store.Next(); ok {
			subjects[triple.Subject] = append(subjects[triple.Subject], triple)
		} else {
			break
		}
	}

	return subjects
}

func splitPrefix(uri string) (base string, name string) {
	index := strings.LastIndex(uri, "#") + 1

	if index > 0 {
		return uri[:index], uri[index:]
	}

	index = strings.LastIndex(uri, "/") + 1

	if index > 0 {
		return uri[:index], uri[index:]
	}

	return "", ""
}

/*
func (graph *Graph) ReadNTriples(reader io.Reader) (err os.Error) {
	buffer := []byte{}
	inputBuffer := new([1024]byte)[:]
	var n int
	var err os.Error
	var line string
	var index int

	for {
		index = bytes.Index("\n", buffer)

		if index < 0 {
			n, err = reader.Read(inputBuffer)
			if err != nil {return err}

			buffer = bytes.Join([][]byte{buffer, inputBuffer}, "")
			index = bytes.Index("\n", buffer)
		}
		

	}
}
*/

func ParseJSONTerm(obj map[string]string) (term *Term, err os.Error) {
	t, type_ok := obj["type"]
	if !type_ok {return nil, os.ErrorString("No type property found")}

	value, value_ok := obj["value"]
	if !value_ok {return nil, os.ErrorString("No value property found")}

	switch t {
		case "uri":
			return CreateResource(value), nil

		case "bnode":
			return CreateNode(value), nil

		case "literal":
			lang, lang_ok := obj["lang"]
			datatype, datatype_ok := obj["datatype"]
			if lang_ok {
				return CreateLiteralWithLanguage(value, lang), nil
			} else if datatype_ok {
				return CreateLiteralWithDatatype(value, CreateResource(datatype)), nil
			} else {
				return CreateLiteral(value), nil
			}

		default:
			return nil, os.ErrorString("Invalid object type")
	}

	return nil, nil
}

func (graph *Graph) ReadJSON(reader io.Reader) (err os.Error) {
	// Yo dawg, I heard you like types, so I put a map in a slice
	// in a map in a map, so you can parse maps while you parse
	// maps, while you parse slices with maps in.
	data := new(map[string]map[string][]map[string]string)

	decoder := json.NewDecoder(reader)
	err = decoder.Decode(data)
	if err != nil {return err}

	var subjTerm *Term
	var predTerm *Term
	var objTerm *Term

	for subj, props := range *data {
		if strings.HasPrefix(subj, "_:") {
			subjTerm = CreateNode(subj[2:])
		} else {
			subjTerm = CreateResource(subj)
		}

		for pred, objs := range props {
			predTerm = CreateResource(pred)

			for i, obj := range objs {
				objTerm, err = ParseJSONTerm(obj)
				if err != nil {return err}
				graph.AddTriple(subjTerm, predTerm, objTerm)
			}
		}
	}
	
	return nil
}

func (graph *Graph) WriteNTriples(writer io.Writer) (err os.Error) {
	graph.store.ResetIter()

	//var buffer []byte

	for {
		if triple, ok := graph.store.Next(); ok {
			n, err := fmt.Fprintf(writer, "%s\n", triple.String())
			if err != nil {return err}
		} else {
			break
		}
	}

	return nil
}

func (graph *Graph) WriteXML(writer io.Writer) (err os.Error) {
	var buffer []byte
	var n int

	subjects := graph.TriplesBySubject()
	pnum := 0
	prefixes := map[string]string{"http://www.w3.org/1999/02/22-rdf-syntax-ns#": "rdf"}

	/*
	for uri, prefix := range graph.prefixes {
		prefixes[uri] = prefix
	}
	*/
	
	for subject, triples := range subjects {
		for i, triple := range triples {
			base, name := splitPrefix(triple.Predicate.Value)

			prefix, ok := prefixes[base]
			if ok {
				if triple.Predicate.prefix == "" {
					triple.Predicate.prefix = prefix
				}
			} else {
				prefix, ok = graph.prefixes[base]
				if !ok {
					prefix = fmt.Sprintf("p%d", pnum)
				}
				triple.Predicate.prefix = prefix
				prefixes[base] = prefix
				pnum++
			}
			triple.Predicate.name = name

			if triple.Object.Language != "" {
				prefixes["http://www.w3.org/XML/1998/namespace"] = "xml"
			}
		}
	}

	n, err = fmt.Fprint(writer, "<rdf:RDF\n")
	if err != nil {return err}

	for uri, prefix := range prefixes {
		n, err = fmt.Fprintf(writer, "  xmlns:%s='%s'\n", prefix, uri)
		if err != nil {return err}
	}

	n, err = fmt.Fprint(writer, ">\n")
	if err != nil {return err}

	for subject, triples := range subjects {
		if subject == nil {continue}

		if subject.Type == TERM_RESOURCE {
			n, err = fmt.Fprintf(writer, "  <rdf:Description rdf:about='%s'>\n", subject.Value)
			if err != nil {return err}
		} else if subject.Type == TERM_NODE {
			n, err = fmt.Fprintf(writer, "  <rdf:Description rdf:nodeID='%s'>\n", subject.Value)
			if err != nil {return err}
		} else {
			return os.NewError("Subject term must be a resource or node")
		}

		for i, triple := range triples {
			predicate := triple.Predicate
			object := triple.Object

			if predicate == nil || object == nil {continue}

			if predicate.Type != TERM_RESOURCE {
				return os.NewError("Predicate term must be a resource")
			}

			n, err = fmt.Fprintf(writer, "    <%s:%s", predicate.prefix, predicate.name)
			if err != nil {return err}
			
			if object.Type == TERM_RESOURCE {
				n, err = fmt.Fprintf(writer, " rdf:resource='%s'/>\n", object.Value)
				if err != nil {return err}
			} else if object.Type == TERM_NODE {
				n, err = fmt.Fprintf(writer, " rdf:nodeID='%s'/>\n", object.Value)
				if err != nil {return err}
			} else if object.Type == TERM_LITERAL {
				if object.Language != "" {
					n, err = fmt.Fprintf(writer, " xml:lang='%s'>%s</a:%s>\n", object.Language, object.Value, predicate.name)
					if err != nil {return err}
				} else if object.Datatype != nil {
					if object.Datatype.Type != TERM_RESOURCE {
						return os.NewError("Object datatype must be a resource")
					}

					n, err = fmt.Fprintf(writer, " rdf:datatype='%s'>%s</a:%s>\n", object.Datatype.Value, object.Value, predicate.name)
					if err != nil {return err}
					
				} else {
					n, err = fmt.Fprintf(writer, ">%s</a:%s>\n", object.Value, predicate.name)
					if err != nil {return err}
				}
			} else {
				return os.NewError("Object term must be a resource, node or literal")
			}
		}
		
		n, err = fmt.Fprint(writer, "  </rdf:Description>\n")
		if err != nil {return err}
	}

	n, err = fmt.Fprint(writer, "</rdf:RDF>\n")
	if err != nil {return err}

	return nil
}
