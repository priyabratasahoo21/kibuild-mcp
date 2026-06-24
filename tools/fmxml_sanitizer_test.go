package tools

import (
	"strings"
	"testing"
)

func TestSanitizeFMXmlSnippet(t *testing.T) {
	// Test case 1: Standard steps with standard clipboard structure
	xmlInput := `<?xml version="1.0" encoding="UTF-8"?>
<fmxmlsnippet type="FMObjectList">
	<Step enable="True" id="89" name="Comment">
		<Text><![CDATA[Test Comment]]></Text>
	</Step>
	<Step enable="True" id="300" name="Set Error Capture">
		<State state="True"/>
	</Step>
	<Step enable="True" id="141" name="Set Variable">
		<Value>
			<Calculation><![CDATA["var_val"]]></Calculation>
		</Value>
		<Repetition>
			<Calculation><![CDATA[1]]></Calculation>
		</Repetition>
		<Name>$testVar</Name>
	</Step>
	<Step enable="True" id="76" name="Set Field">
		<Calculation><![CDATA["field_val"]]></Calculation>
		<Field name="FieldName" table="TableName"></Field>
	</Step>
	<Step enable="True" id="1" name="Perform Script">
		<Script name="TargetScript"></Script>
		<Parameter>
			<Calculation><![CDATA["param"]]></Calculation>
		</Parameter>
	</Step>
	<Step enable="True" id="103" name="Exit Script">
		<Value>
			<Calculation><![CDATA["exit_val"]]></Calculation>
		</Value>
	</Step>
</fmxmlsnippet>`

	sanitized, err := SanitizeFMXmlSnippet(xmlInput)
	if err != nil {
		t.Fatalf("SanitizeFMXmlSnippet failed: %v", err)
	}

	expectedLines := []string{
		"# Test Comment",
		"Set Error Capture [ On ]",
		"Set Variable [ $testVar ; Value: \"var_val\" ]",
		"Set Field [ TableName::FieldName ; \"field_val\" ]",
		"Perform Script [ \"TargetScript\" ; Parameter: \"param\" ]",
		"Exit Script [ Text Result: \"exit_val\" ]",
	}

	for _, expected := range expectedLines {
		if !strings.Contains(sanitized, expected) {
			t.Errorf("Expected sanitized output to contain '%s', got:\n%s", expected, sanitized)
		}
	}

	// Test case 2: Nested calculation in DDR structure
	ddrXmlInput := `<?xml version="1.0" encoding="UTF-8"?>
<fmxmlsnippet type="FMObjectList">
	<Step enable="True" id="141" name="Set Variable">
		<ParameterValues membercount="1">
			<Parameter type="Variable">
				<value>
					<Calculation datatype="1" position="1">
						<Calculation>
							<Text><![CDATA[NestedCalcResult]]></Text>
						</Calculation>
					</Calculation>
				</value>
				<Name value="$setup"></Name>
				<repetition>
					<Calculation datatype="1" position="2">
						<Calculation>
							<Text><![CDATA[1]]></Text>
						</Calculation>
					</Calculation>
				</repetition>
			</Parameter>
		</ParameterValues>
	</Step>
</fmxmlsnippet>`

	sanitizedDdr, err := SanitizeFMXmlSnippet(ddrXmlInput)
	if err != nil {
		t.Fatalf("SanitizeFMXmlSnippet for DDR failed: %v", err)
	}

	expectedDdr := "Set Variable [ $setup ; Value: NestedCalcResult ]"
	if !strings.Contains(sanitizedDdr, expectedDdr) {
		t.Errorf("Expected DDR output to contain '%s', got:\n%s", expectedDdr, sanitizedDdr)
	}

	// Test case 3: DDR reference elements (ScriptReference, FieldReference, TableOccurrenceReference)
	refXmlInput := `<?xml version="1.0" encoding="UTF-8"?>
<fmxmlsnippet type="FMObjectList">
	<Step enable="True" id="1" name="Perform Script">
		<ScriptReference name="DdrScript"></ScriptReference>
		<Parameter>
			<Calculation><![CDATA["ddr_param"]]></Calculation>
		</Parameter>
	</Step>
	<Step enable="True" id="76" name="Set Field">
		<Calculation><![CDATA["ddr_field_val"]]></Calculation>
		<FieldReference name="DdrFieldName">
			<TableOccurrenceReference name="DdrTableName"></TableOccurrenceReference>
		</FieldReference>
	</Step>
</fmxmlsnippet>`

	sanitizedRef, err := SanitizeFMXmlSnippet(refXmlInput)
	if err != nil {
		t.Fatalf("SanitizeFMXmlSnippet for references failed: %v", err)
	}

	expectedRefs := []string{
		"Perform Script [ \"DdrScript\" ; Parameter: \"ddr_param\" ]",
		"Set Field [ DdrTableName::DdrFieldName ; \"ddr_field_val\" ]",
	}

	for _, expected := range expectedRefs {
		if !strings.Contains(sanitizedRef, expected) {
			t.Errorf("Expected reference output to contain '%s', got:\n%s", expected, sanitizedRef)
		}
	}
}
